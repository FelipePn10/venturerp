//go:build integration

package cost_uc_test

import (
	"math"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cost_uc"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	scentity "github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity"
	routingRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	standardCostRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/standard_cost"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Verifies the enterprise+ conversion cost: each operation is charged at its OWN
// work-center rate (no naive average) using the rich, machine × labor time model.
func TestIntegration_CostRollup_PerWorkCenterRichTime(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := cost_uc.New(standardCostRepo.New(q, pool)).WithRouting(routingRepo.New(q))
	rRepo := routingRepo.New(q)
	scRepo := standardCostRepo.New(q, pool)
	ctx := testutil.TenantContext(t, pool)
	uid := uuid.New()

	// Work center (machine type) + its machine/labor hourly rates.
	wcCode := testutil.UniqueCode()
	var wcID int64
	if err := pool.QueryRow(ctx,
		"INSERT INTO machine_types (code, name, type, requires_operator, created_by, enterprise_id) VALUES ($1,'CT Custo','CUT',false,$2,$3) RETURNING id",
		wcCode, uid, testutil.EnterpriseID(t, ctx)).Scan(&wcID); err != nil {
		t.Fatalf("seed machine_type: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM machine_types WHERE id = $1", wcID)

	if _, err := scRepo.UpsertWorkCenterCost(ctx, &scentity.WorkCenterCost{
		WorkCenterID: wcID, CostPerHour: 100, MachineCostPerHour: 100, LaborCostPerHour: 50,
		Currency: "BRL", UpdatedBy: uid,
	}); err != nil {
		t.Fatalf("UpsertWorkCenterCost: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM work_center_costs WHERE work_center_id = $1", wcID)

	// Operation (MIN): setup 60, run 30/10pç, labor 20/10pç, crew 2, default WC.
	// Unique code (not NextOperationCode) to avoid collisions with parallel test packages.
	opCode := testutil.UniqueCode()
	op, _ := routingentity.NewOperation(opCode, "Corte", nil, routingentity.OriginInternal, &wcID, 0, 0, uid)
	op.SetupTime, op.RunTime, op.LaborTime = 60, 30, 20
	op.RunBaseQty, op.CrewSize, op.TimeUnit = 10, 2, routingentity.TimeUnitMinute
	createdOp, err := rRepo.CreateOperation(ctx, op)
	if err != nil {
		t.Fatalf("CreateOperation: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM operations WHERE id = $1", createdOp.ID)

	// Item + standard route + one (inherited) operation.
	itemCode := testutil.UniqueCode()
	testutil.SeedItem(t, pool, ctx, itemCode, uid)
	defer testutil.Exec(t, pool, "DELETE FROM items WHERE code = $1", itemCode)

	rc := testutil.UniqueCode()
	rt, _ := routingentity.NewManufacturingRoute(rc, itemCode, nil, 1, nil, true, nil, nil, uid)
	route, err := rRepo.CreateRoute(ctx, rt)
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM manufacturing_routes WHERE id = $1", route.ID)
	ro, _ := routingentity.NewRouteOperation(route.ID, 10, createdOp.ID, nil, nil, nil, nil)
	if _, err := rRepo.AddRouteOperation(ctx, ro); err != nil {
		t.Fatalf("AddRouteOperation: %v", err)
	}

	defer testutil.Exec(t, pool, "DELETE FROM item_standard_costs WHERE item_code = $1", itemCode)
	defer testutil.Exec(t, pool, "DELETE FROM cost_rollup_log WHERE item_code = $1", itemCode)

	res, err := uc.RollUp(ctx, request.CostRollupDTO{ItemCode: itemCode, Mask: "", CalculatedBy: uid.String()})
	if err != nil {
		t.Fatalf("RollUp: %v", err)
	}

	// Esperado (qtd=1, em horas):
	//   HorasMáquina = setup 1h + run 0,5h×ceil(1/10)=0,5 → 1,5h × R$100 = 150,00
	//   HorasHomem   = (setup 1h + labor 0,3333h×1) × equipe 2 = 2,6667h × R$50 = 133,33
	//   conversão    = 283,33
	//
	// A conversão agora sai ABERTA por componente (migração 000373): antes os três
	// números vinham somados no campo `labor_cost`, e não havia como saber quanto era
	// preparação. A conferência aqui é dupla: cada componente no seu lugar E a soma
	// preservada — é a soma que prova que a separação não perdeu nem inventou custo.
	const (
		taxaMaquina = 100.0
		taxaHomem   = 50.0
	)
	setupEsperado := 1.0*taxaMaquina + (1.0*2)*taxaHomem // 1h máquina + 1h × equipe 2 homem = 200,00
	maquinaEsperada := 0.5 * taxaMaquina                 // 0,5h de produção = 50,00
	homemEsperado := (20.0 / 60.0 * 2) * taxaHomem       // 0,3333h × equipe 2 = 33,33
	conversaoEsperada := 1.5*taxaMaquina + (1.0+20.0/60.0)*2*taxaHomem

	if math.Abs(res.SetupCost-setupEsperado) > 0.05 {
		t.Fatalf("setup_cost = %.4f, esperado %.4f", res.SetupCost, setupEsperado)
	}
	if math.Abs(res.MachineCost-maquinaEsperada) > 0.05 {
		t.Fatalf("machine_cost = %.4f, esperado %.4f", res.MachineCost, maquinaEsperada)
	}
	if math.Abs(res.LaborCost-homemEsperado) > 0.05 {
		t.Fatalf("labor_cost = %.4f, esperado %.4f (só mão de obra direta)", res.LaborCost, homemEsperado)
	}
	// A soma dos componentes tem de reproduzir a conversão de antes, ao centavo.
	soma := res.SetupCost + res.MachineCost + res.LaborCost
	if math.Abs(soma-conversaoEsperada) > 0.05 {
		t.Fatalf("setup+máquina+homem = %.4f, esperado %.4f — a separação perdeu ou inventou custo", soma, conversaoEsperada)
	}
	if math.Abs(res.TotalCost-conversaoEsperada) > 0.05 {
		t.Fatalf("total_cost = %.4f, esperado %.4f (item sem material)", res.TotalCost, conversaoEsperada)
	}
	t.Logf("setup=%.2f máquina=%.2f homem=%.2f total=%.2f — conversão por CT aberta OK",
		res.SetupCost, res.MachineCost, res.LaborCost, res.TotalCost)

	// Diluição do setup: lote de referência 10 espalha a preparação por 10 unidades.
	// run_base_qty=10 ⇒ um ciclo cobre o lote, então o custo do lote ÷ 10.
	resLot, err := uc.RollUp(ctx, request.CostRollupDTO{ItemCode: itemCode, Mask: "", LotSize: 10, CalculatedBy: uid.String()})
	if err != nil {
		t.Fatalf("RollUp (lote): %v", err)
	}
	if math.Abs(resLot.TotalCost-conversaoEsperada/10) > 0.05 {
		t.Fatalf("total (lote=10) = %.4f, esperado %.4f", resLot.TotalCost, conversaoEsperada/10)
	}
	// É o SETUP que a diluição tem de reduzir dez vezes — a produção por peça não
	// muda com o lote. Conferir só o total esconderia um erro que trocasse os dois.
	if math.Abs(resLot.SetupCost-setupEsperado/10) > 0.05 {
		t.Fatalf("setup (lote=10) = %.4f, esperado %.4f", resLot.SetupCost, setupEsperado/10)
	}
	if resLot.TotalCost >= res.TotalCost {
		t.Errorf("a diluição pelo lote deveria baixar o custo unitário: lote=10 %.2f contra lote=1 %.2f", resLot.TotalCost, res.TotalCost)
	}
	t.Logf("total(lote=10)=%.2f setup=%.2f — setup diluído OK", resLot.TotalCost, resLot.SetupCost)
}
