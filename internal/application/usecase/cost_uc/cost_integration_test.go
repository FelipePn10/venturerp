//go:build integration

package cost_uc_test

import (
	"context"
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
	uc := cost_uc.New(standardCostRepo.New(q)).WithRouting(routingRepo.New(q))
	rRepo := routingRepo.New(q)
	scRepo := standardCostRepo.New(q)
	ctx := context.Background()
	uid := uuid.New()

	// Work center (machine type) + its machine/labor hourly rates.
	wcCode := testutil.UniqueCode()
	var wcID int64
	if err := pool.QueryRow(ctx,
		"INSERT INTO machine_types (code, name, type, requires_operator, created_by) VALUES ($1,'CT Custo','CUT',false,$2) RETURNING id",
		wcCode, uid).Scan(&wcID); err != nil {
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
	testutil.Exec(t, pool, "INSERT INTO items (code, warehouse_code, created_by) VALUES ($1,$2,$3)", itemCode, itemCode, uid)
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

	// Expected (qty=1, hours):
	//   MachineHours = setup 1h + run 0.5h×ceil(1/10)=0.5 → 1.5h × R$100 = 150.00
	//   LaborHours   = (setup 1h + labor 0.3333h×1) × crew 2 = 2.6667h × R$50 = 133.33
	//   labor_cost   = 283.33
	want := 1.5*100 + (1.0+20.0/60.0)*2*50
	if math.Abs(res.LaborCost-want) > 0.05 {
		t.Fatalf("labor_cost = %.4f, want %.4f", res.LaborCost, want)
	}
	t.Logf("labor_cost=%.2f (machine 150 + labor 133.33) — per-CT rich costing OK", res.LaborCost)

	// Setup amortization: a reference lot of 10 spreads the one-off setup over 10 units.
	// run_base_qty=10 ⇒ one run cycle covers the lot, so the whole lot cost ÷ 10.
	resLot, err := uc.RollUp(ctx, request.CostRollupDTO{ItemCode: itemCode, Mask: "", LotSize: 10, CalculatedBy: uid.String()})
	if err != nil {
		t.Fatalf("RollUp (lot): %v", err)
	}
	wantLot := want / 10
	if math.Abs(resLot.LaborCost-wantLot) > 0.05 {
		t.Fatalf("labor_cost (lot=10) = %.4f, want %.4f", resLot.LaborCost, wantLot)
	}
	if resLot.LaborCost >= res.LaborCost {
		t.Errorf("lot amortization should lower unit cost: lot=10 %.2f vs lot=1 %.2f", resLot.LaborCost, res.LaborCost)
	}
	t.Logf("labor_cost(lot=10)=%.2f — setup amortized OK", resLot.LaborCost)
}
