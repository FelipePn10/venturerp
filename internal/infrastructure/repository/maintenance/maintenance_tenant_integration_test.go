//go:build integration

package maintenance_test

import (
	"context"
	"strconv"
	"testing"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/maintenance/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	maintenance "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/maintenance"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func maintenanceTenantContext(enterpriseID int64) context.Context {
	user := &appsecurity.AuthUser{ID: uuid.NewString(), EnterpriseID: enterpriseID}
	return context.WithValue(context.Background(), contextkey.UserKey, user)
}

func TestMaintenancePlansAreTenantIsolatedAndAudited(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := uuid.New()
	var enterpriseA, enterpriseB int64
	codeA := int64(1_700_000_000 + testutil.UniqueCode()%100_000_000)
	codeB := codeA + 1
	if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name) VALUES($1,'Manutenção A') RETURNING id`, codeA).Scan(&enterpriseA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name) VALUES($1,'Manutenção B') RETURNING id`, codeB).Scan(&enterpriseB); err != nil {
		t.Fatal(err)
	}

	createMachine := func(enterpriseID int64) (int64, int64) {
		machineTypeCode, machineCode := testutil.UniqueCode(), testutil.UniqueCode()
		var machineTypeID, machineID int64
		if err := pool.QueryRow(base, `INSERT INTO machine_types(code,name,type,created_by,enterprise_id) VALUES($1,'Centro manutenção','CUT',$2,$3) RETURNING id`, machineTypeCode, actor, enterpriseID).Scan(&machineTypeID); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(base, `INSERT INTO machines(code,name,machine_type_code,capacity,capacity_unit,capacity_period,efficiency_rate,created_by,enterprise_id) VALUES($1,'Máquina manutenção',$2,8,'UN','HORA',1,$3,$4) RETURNING id`, machineCode, machineTypeCode, actor, enterpriseID).Scan(&machineID); err != nil {
			t.Fatal(err)
		}
		return machineTypeID, machineID
	}
	wcA, machineA := createMachine(enterpriseA)
	wcB, machineB := createMachine(enterpriseB)
	t.Cleanup(func() {
		_, _ = pool.Exec(base, `DELETE FROM maintenance_orders WHERE plan_id IN (SELECT id FROM maintenance_plans WHERE machine_id=ANY($1))`, []int64{machineA, machineB})
		_, _ = pool.Exec(base, `DELETE FROM maintenance_plans WHERE machine_id=ANY($1)`, []int64{machineA, machineB})
		_, _ = pool.Exec(base, `DELETE FROM machines WHERE id=ANY($1)`, []int64{machineA, machineB})
		_, _ = pool.Exec(base, `DELETE FROM machine_types WHERE id=ANY($1)`, []int64{wcA, wcB})
		_, _ = pool.Exec(base, `DELETE FROM enterprise WHERE id=ANY($1)`, []int64{enterpriseA, enterpriseB})
	})

	repo := maintenance.New(sqlc.New(pool), pool)
	planA, err := entity.NewMaintenancePlan(machineA, &wcA, "Preventiva A", entity.FrequencyMonthly, 30, 4, actor)
	if err != nil {
		t.Fatal(err)
	}
	createdA, err := repo.CreatePlan(maintenanceTenantContext(enterpriseA), planA)
	if err != nil {
		t.Fatal(err)
	}
	planB, _ := entity.NewMaintenancePlan(machineB, &wcB, "Preventiva B", entity.FrequencyMonthly, 30, 4, actor)
	if _, err = repo.CreatePlan(maintenanceTenantContext(enterpriseB), planB); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetPlanByID(maintenanceTenantContext(enterpriseB), createdA.ID); err == nil {
		t.Fatal("tenant B conseguiu consultar plano do tenant A")
	}
	listA, err := repo.ListPlans(maintenanceTenantContext(enterpriseA), true)
	if err != nil || len(listA) != 1 || listA[0].ID != createdA.ID {
		t.Fatalf("catálogo do tenant A incorreto: plans=%+v err=%v", listA, err)
	}
	if err = repo.DeactivatePlan(maintenanceTenantContext(enterpriseA), createdA.ID); err != nil {
		t.Fatal(err)
	}
	var inserts, updates int
	rowKey := strconv.FormatInt(createdA.ID, 10)
	if err = pool.QueryRow(base, `SELECT COUNT(*) FILTER(WHERE operation='INSERT'),COUNT(*) FILTER(WHERE operation='UPDATE') FROM operational_mutation_audit WHERE table_name='maintenance_plans' AND row_key=$1`, rowKey).Scan(&inserts, &updates); err != nil {
		t.Fatal(err)
	}
	if inserts < 1 || updates < 1 {
		t.Fatalf("auditoria antes/depois ausente: inserts=%d updates=%d", inserts, updates)
	}
	if _, err = pool.Exec(base, `UPDATE operational_mutation_audit SET row_key='alterado' WHERE table_name='maintenance_plans' AND row_key=$1`, rowKey); err == nil {
		t.Fatal("auditoria permitiu alteração")
	}
}
