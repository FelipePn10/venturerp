//go:build integration

package routing_test

import (
	"context"
	"math"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routingrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func approxEq(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

// Exercises the rich-time columns end-to-end against a real Postgres: verifies the
// sqlc scan column ordering for operations/route_operations and the EffTime
// resolution (unit conversion + override inheritance) done in the repository.
func TestIntegration_Routing_RichTimeRoundTrip(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := routingrepo.New(q)
	ctx := context.Background()
	uid := uuid.New()

	// 1. Operation with a rich time model measured in MINUTES.
	// Unique code (not NextOperationCode) to avoid collisions with parallel test packages.
	opCode := testutil.UniqueCode()
	op, err := entity.NewOperation(opCode, "Corte laser", nil, entity.OriginInternal, nil, 0, 0, uid)
	if err != nil {
		t.Fatalf("NewOperation: %v", err)
	}
	op.RunTime = 30 // min per 10 pcs
	op.SetupTime = 60
	op.RunBaseQty = 10
	op.CrewSize = 2
	op.TimeUnit = entity.TimeUnitMinute
	created, err := repo.CreateOperation(ctx, op)
	if err != nil {
		t.Fatalf("CreateOperation: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM operations WHERE id = $1", created.ID)

	// Round-trip the operation scan.
	gotOp, err := repo.GetOperationByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetOperationByID: %v", err)
	}
	if !approxEq(gotOp.RunTime, 30) || !approxEq(gotOp.RunBaseQty, 10) ||
		!approxEq(gotOp.CrewSize, 2) || gotOp.TimeUnit != entity.TimeUnitMinute {
		t.Fatalf("operation round-trip mismatch: %+v", gotOp)
	}

	// 2. Minimal item to hang the route on (items has no FKs; PK id, unique code).
	itemCode := testutil.UniqueCode()
	testutil.Exec(t, pool, "INSERT INTO items (code, warehouse_code, created_by) VALUES ($1, $2, $3)", itemCode, itemCode, uid)
	defer testutil.Exec(t, pool, "DELETE FROM items WHERE code = $1", itemCode)

	// 3. Route + a route operation that INHERITS the operation defaults.
	routeCode := testutil.UniqueCode()
	rt, err := entity.NewManufacturingRoute(routeCode, itemCode, nil, 1, nil, true, nil, nil, uid)
	if err != nil {
		t.Fatalf("NewManufacturingRoute: %v", err)
	}
	route, err := repo.CreateRoute(ctx, rt)
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM manufacturing_routes WHERE id = $1", route.ID) // cascades route_operations

	ro, err := entity.NewRouteOperation(route.ID, 1, created.ID, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewRouteOperation: %v", err)
	}
	if _, err := repo.AddRouteOperation(ctx, ro); err != nil {
		t.Fatalf("AddRouteOperation: %v", err)
	}

	ops, err := repo.GetRouteOperations(ctx, route.ID)
	if err != nil {
		t.Fatalf("GetRouteOperations: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("want 1 route op, got %d", len(ops))
	}
	eff := ops[0].EffTime
	// Inherited from op, converted to hours: setup 60min=1h, run 30min=0.5h, base 10, crew 2.
	if !approxEq(eff.Setup, 1) || !approxEq(eff.Run, 0.5) ||
		!approxEq(eff.RunBaseQty, 10) || !approxEq(eff.CrewSize, 2) {
		t.Fatalf("inherited EffTime mismatch: %+v", eff)
	}
	// MachineHours(10) = setup + run*ceil(10/10) = 1 + 0.5 = 1.5.
	if !approxEq(eff.MachineHours(10), 1.5) {
		t.Errorf("MachineHours(10) = %v, want 1.5", eff.MachineHours(10))
	}
	// LaborHours(10) = (setup + run*1) * crew = 1.5 * 2 = 3.
	if !approxEq(eff.LaborHours(10), 3) {
		t.Errorf("LaborHours(10) = %v, want 3", eff.LaborHours(10))
	}

	// End-to-end: the DB-resolved EffTime must flow into the shared CPM. A single
	// operation with no network edges → lead time = its own LeadTimeHours(10) = 1.5.
	if lt := entity.CriticalPath(ops, nil, 10).TotalHours; !approxEq(lt, 1.5) {
		t.Errorf("CriticalPath lead time = %v h, want 1.5", lt)
	}

	// 4. Override the run time on the route op (in HOURS) → must beat inheritance.
	runOverride := 2.0
	unit := entity.TimeUnitHour
	ops[0].RunTime = &runOverride
	ops[0].TimeUnit = &unit
	if _, err := repo.UpdateRouteOperation(ctx, ops[0]); err != nil {
		t.Fatalf("UpdateRouteOperation: %v", err)
	}
	ops2, err := repo.GetRouteOperations(ctx, route.ID)
	if err != nil {
		t.Fatalf("GetRouteOperations after override: %v", err)
	}
	if !approxEq(ops2[0].EffTime.Run, 2) {
		t.Errorf("overridden run = %v h, want 2", ops2[0].EffTime.Run)
	}
}
