//go:build integration

package machine

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func TestCreateTypeAndMachineUseAuthenticatedTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := uuid.New()
	code := int64(1_000_000_000 + testutil.UniqueCode()%500_000_000)
	var enterpriseID int64
	testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password) VALUES($1,'Máquinas',$2,'x')`, actor, actor.String()+"@test.local")
	if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Máquinas',$2) RETURNING id`, code, actor).Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	repo := NewMachineRepositorySQLC(sqlc.New(pool))
	typeCode, machineCode := testutil.UniqueCode(), testutil.UniqueCode()
	t.Cleanup(func() {
		_, _ = pool.Exec(base, `DELETE FROM machines WHERE code=$1`, machineCode)
		_, _ = pool.Exec(base, `DELETE FROM machine_types WHERE code=$1`, typeCode)
		_, _ = pool.Exec(base, `DELETE FROM enterprise WHERE id=$1`, enterpriseID)
		_, _ = pool.Exec(base, `DELETE FROM users WHERE id=$1`, actor)
	})

	if _, err := repo.CreateType(ctx, &entity.MachineType{Code: typeCode, Name: "Corte", Type: types.MachineCut, IsActive: true, CreatedBy: actor}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(ctx, &entity.Machine{Code: machineCode, Name: "Serra", MachineTypeCode: typeCode, Capacity: 8, CapacityUnit: types.Units, CapacityPeriod: types.Hour, EfficiencyRate: 1, IsActive: true, CreatedBy: actor}); err != nil {
		t.Fatal(err)
	}
	var typeTenant, machineTenant int64
	if err := pool.QueryRow(base, `SELECT enterprise_id FROM machine_types WHERE code=$1`, typeCode).Scan(&typeTenant); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `SELECT enterprise_id FROM machines WHERE code=$1`, machineCode).Scan(&machineTenant); err != nil {
		t.Fatal(err)
	}
	if typeTenant != enterpriseID || machineTenant != enterpriseID {
		t.Fatalf("tenant incorreto: tipo=%d máquina=%d esperado=%d", typeTenant, machineTenant, enterpriseID)
	}
}
