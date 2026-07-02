//go:build integration

package routing_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routingrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Exercises alternative resources per route operation: adding a primary + an
// alternative, listing them, and switching the primary — which must mirror onto
// the route operation's effective work center (used by cost/CRP/lead-time).
func TestIntegration_Routing_AlternativeResources(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := routingrepo.New(q)
	ctx := context.Background()
	uid := uuid.New()

	// Two work centers.
	var wc1, wc2 int64
	if err := pool.QueryRow(ctx, "INSERT INTO machine_types (code,name,type,requires_operator,created_by) VALUES ($1,'CT-A','CUT',false,$2) RETURNING id", testutil.UniqueCode(), uid).Scan(&wc1); err != nil {
		t.Fatalf("seed wc1: %v", err)
	}
	if err := pool.QueryRow(ctx, "INSERT INTO machine_types (code,name,type,requires_operator,created_by) VALUES ($1,'CT-B','CUT',false,$2) RETURNING id", testutil.UniqueCode(), uid).Scan(&wc2); err != nil {
		t.Fatalf("seed wc2: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM machine_types WHERE id IN ($1,$2)", wc1, wc2)

	// Operation + item + route + one route op.
	opCode := testutil.UniqueCode()
	op, _ := entity.NewOperation(opCode, "Corte", nil, entity.OriginInternal, &wc1, 1, 0, uid)
	createdOp, err := repo.CreateOperation(ctx, op)
	if err != nil {
		t.Fatalf("CreateOperation: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM operations WHERE id = $1", createdOp.ID)

	itemCode := testutil.UniqueCode()
	testutil.Exec(t, pool, "INSERT INTO items (code, warehouse_code, created_by) VALUES ($1,$2,$3)", itemCode, itemCode, uid)
	defer testutil.Exec(t, pool, "DELETE FROM items WHERE code = $1", itemCode)

	rc := testutil.UniqueCode()
	rt, _ := entity.NewManufacturingRoute(rc, itemCode, nil, 1, nil, true, nil, nil, uid)
	route, err := repo.CreateRoute(ctx, rt)
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM manufacturing_routes WHERE id = $1", route.ID)
	ro, _ := entity.NewRouteOperation(route.ID, 10, createdOp.ID, nil, nil, nil, nil)
	addedOp, err := repo.AddRouteOperation(ctx, ro)
	if err != nil {
		t.Fatalf("AddRouteOperation: %v", err)
	}

	// Primary resource on WC1 → mirrors onto route_operations.work_center_id.
	resA, err := repo.AddRouteOpResource(ctx, &entity.RouteOpResource{RouteOperationID: addedOp.ID, WorkCenterID: wc1, Priority: 1, TimeFactor: 1})
	if err != nil {
		t.Fatalf("AddRouteOpResource A: %v", err)
	}
	if _, err := repo.SetRouteOpResourcePrimary(ctx, resA.ID, addedOp.ID, wc1); err != nil {
		t.Fatalf("SetPrimary A: %v", err)
	}
	// Alternative on WC2 (20% slower).
	if _, err := repo.AddRouteOpResource(ctx, &entity.RouteOpResource{RouteOperationID: addedOp.ID, WorkCenterID: wc2, Priority: 2, TimeFactor: 1.2}); err != nil {
		t.Fatalf("AddRouteOpResource B: %v", err)
	}

	list, err := repo.ListResourcesByRouteOp(ctx, addedOp.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("ListResources = %d err=%v, want 2", len(list), err)
	}
	if !list[0].IsPrimary || list[0].WorkCenterID != wc1 {
		t.Errorf("primary should be WC1 first, got %+v", list[0])
	}

	// Effective work center of the operation now = WC1.
	if wc := effectiveWC(t, repo, route.ID); wc != wc1 {
		t.Fatalf("effective WC = %d, want wc1 %d", wc, wc1)
	}

	// Switch primary to the alternative (WC2) → effective WC follows.
	resB := list[1]
	if _, err := repo.SetRouteOpResourcePrimary(ctx, resB.ID, addedOp.ID, wc2); err != nil {
		t.Fatalf("SetPrimary B: %v", err)
	}
	if wc := effectiveWC(t, repo, route.ID); wc != wc2 {
		t.Fatalf("after switch, effective WC = %d, want wc2 %d", wc, wc2)
	}
}

func effectiveWC(t *testing.T, repo interface {
	GetRouteOperations(ctx context.Context, routeID int64) ([]*entity.RouteOperation, error)
}, routeID int64) int64 {
	t.Helper()
	ops, err := repo.GetRouteOperations(context.Background(), routeID)
	if err != nil || len(ops) == 0 {
		t.Fatalf("GetRouteOperations: %v", err)
	}
	if ops[0].EffectiveWorkCenterID == nil {
		t.Fatal("effective work center is nil")
	}
	return *ops[0].EffectiveWorkCenterID
}
