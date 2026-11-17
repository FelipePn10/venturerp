//go:build integration

package service

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	machineentity "github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	machinerepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/machine"
	mrprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

// O caso relatado, ponta a ponta no banco: o roteiro tem o tempo da etapa e a
// produtividade foi gravada SEM tempo. Antes o MRP recusava com "produtividade
// inválida"; agora reserva a máquina pelo tempo do roteiro. E, sem máquina no
// centro, a mensagem diz o código comercial do item e que o tempo vem do roteiro.
func TestMRPUsaTempoDoRoteiroComProdutividadeSemTempo(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	actor := uuid.New()
	code := testutil.UniqueCode()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO users(id,name,email,password) VALUES($1,'Roteiro test',$2,'x')`, actor, actor.String()+"@test.local")
	var enterprise int64
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Roteiro test',$2) RETURNING id`, code%1000000000, actor).Scan(&enterprise); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testutil.ApagarEmpresas(t, pool, enterprise)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, actor)
	})
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterprise})
	q := sqlc.New(pool)
	machines := machinerepo.NewMachineRepositorySQLC(q, pool)
	mt, err := machines.CreateType(ctx, &machineentity.MachineType{Code: code, Name: "Laser", Type: types.MachineCut, IsActive: true, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	hours := 8.0
	if _, err = machines.Create(ctx, &machineentity.Machine{Code: code, Name: "Laser 1", MachineTypeCode: mt.Code, Capacity: 1, CapacityUnit: types.Pieces,
		CapacityPeriod: types.Hour, EfficiencyRate: 1, AvailableHoursPerDay: &hours, IsActive: true, CreatedBy: actor}); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO items(code,business_code,warehouse_code,health,created_by,enterprise_id) VALUES($1,'RN-01001',0,'ATIVO',$2,$3)`, code, actor, enterprise)
	// Produtividade só com a máquina: tempo ZERO (o tempo está no roteiro).
	if _, err = machines.CreateItemMachineTime(ctx, &machineentity.ItemMachineTime{ItemCode: code, MachineCode: code, ProductionTime: 0,
		ProductionTimeUnit: types.Hour, ProductionBaseQty: 1, Priority: 1, TimeBasis: "CYCLE"}); err != nil {
		t.Fatal(err)
	}
	// Roteiro: uma etapa de 0,5 h por peça no centro da máquina.
	var routeID, operationID int64
	if err = pool.QueryRow(ctx, `INSERT INTO manufacturing_routes(code,item_code,is_standard,created_by,enterprise_id) VALUES($1,$1,true,$2,$3) RETURNING id`, code, actor, enterprise).Scan(&routeID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO operations(code,name,default_work_center_id,created_by,enterprise_id) VALUES($1,'Corte',$2,$3,$4) RETURNING id`, code, mt.ID, actor, enterprise).Scan(&operationID); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO route_operations(route_id,sequence,operation_id,run_time,run_time_base_qty,time_unit) VALUES($1,10,$2,0.5,1,'HORA')`, routeID, operationID)
	exec(`INSERT INTO production_plans(code,name,created_by,enterprise_id) VALUES($1,'Plano roteiro',$2,$3)`, code, actor, enterprise)
	var sugestao int64
	if err = pool.QueryRow(ctx, `INSERT INTO mrp_planned_suggestions(plan_code,item_code,quantity,need_date,start_date,order_type,enterprise_id)
		VALUES($1,$1,4,'2026-09-14','2026-09-14','FABRICACAO',$2) RETURNING code`, code, enterprise).Scan(&sugestao); err != nil {
		t.Fatal(err)
	}

	svc := &MRPServiceImpl{MRPRepo: mrprepo.NewMRPCalculationRepositorySQLC(q, pool)}
	if err = svc.processMachineIntegration(ctx, code, []int64{code}); err != nil {
		t.Fatalf("o MRP recusou a produtividade sem tempo com tempo no roteiro: %v", err)
	}
	var minutos float64
	if err = pool.QueryRow(ctx, `SELECT production_minutes FROM mrp_machine_allocations WHERE suggestion_code=$1`, sugestao).Scan(&minutos); err != nil {
		t.Fatal(err)
	}
	if math.Abs(minutos-120) > 1e-6 { // 4 peças × 0,5 h
		t.Fatalf("o tempo deveria vir do roteiro (120 min), veio %v", minutos)
	}

	// Sem a máquina cadastrada para o item: a mensagem nova.
	exec(`UPDATE item_machine_times SET is_active=false WHERE item_code=$1 AND enterprise_id=$2`, code, enterprise)
	err = svc.processMachineIntegration(ctx, code, []int64{code})
	if err == nil || !strings.Contains(err.Error(), "cadastre o RN-01001 na produtividade de uma máquina de cada centro do roteiro (o tempo vem do roteiro)") {
		t.Fatalf("mensagem sem máquina: %v", err)
	}
}
