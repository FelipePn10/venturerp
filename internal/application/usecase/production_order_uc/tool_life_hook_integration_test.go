//go:build integration

package production_order_uc_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	toolentity "github.com/FelipePn10/panossoerp/internal/domain/tool/entity"
	routingRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	toolRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/tool"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Drives the real shop-floor hook: completing (DONE) a production-order operation
// consumes the useful life of the tools linked to its route operation and surfaces
// a replacement alert when the limit is crossed.
func TestIntegration_ToolLifeConsumedOnOperationDone(t *testing.T) {
	q, pool := testutil.Queries(t)
	rRepo := routingRepo.New(q)
	tRepo := toolRepo.New(q)
	uc := &production_order_uc.OrderOperationsUseCase{Q: q}
	ctx := context.Background()
	uid := uuid.New()

	// Work center + operation + item + route + route op.
	var wcID int64
	if err := pool.QueryRow(ctx, "INSERT INTO machine_types (code,name,type,requires_operator,created_by) VALUES ($1,'CT-Tool','PRESS',false,$2) RETURNING id", testutil.UniqueCode(), uid).Scan(&wcID); err != nil {
		t.Fatalf("seed wc: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM machine_types WHERE id = $1", wcID)

	op, _ := routingentity.NewOperation(testutil.UniqueCode(), "Estampar", nil, routingentity.OriginInternal, &wcID, 1, 0, uid)
	createdOp, err := rRepo.CreateOperation(ctx, op)
	if err != nil {
		t.Fatalf("CreateOperation: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM operations WHERE id = $1", createdOp.ID)

	itemCode := testutil.UniqueCode()
	testutil.Exec(t, pool, "INSERT INTO items (code, warehouse_code, created_by) VALUES ($1,$2,$3)", itemCode, itemCode, uid)
	defer testutil.Exec(t, pool, "DELETE FROM items WHERE code = $1", itemCode)

	rt, _ := routingentity.NewManufacturingRoute(testutil.UniqueCode(), itemCode, nil, 1, nil, true, nil, nil, uid)
	route, err := rRepo.CreateRoute(ctx, rt)
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM manufacturing_routes WHERE id = $1", route.ID)
	ro, _ := routingentity.NewRouteOperation(route.ID, 10, createdOp.ID, nil, nil, nil, nil)
	routeOp, err := rRepo.AddRouteOperation(ctx, ro)
	if err != nil {
		t.Fatalf("AddRouteOperation: %v", err)
	}

	// Tool with a 5-stroke life, linked to the route operation.
	tl, _ := toolentity.NewTool(testutil.UniqueCode(), "Matriz", "MATRIZ", toolentity.LifeStrokes, 5, 0, uid)
	createdTool, err := tRepo.CreateTool(ctx, tl)
	if err != nil {
		t.Fatalf("CreateTool: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM tools WHERE id = $1", createdTool.ID)
	if _, err := tRepo.AddRouteOpTool(ctx, &toolentity.RouteOpTool{RouteOperationID: routeOp.ID, ToolID: createdTool.ID, QtyRequired: 1}); err != nil {
		t.Fatalf("AddRouteOpTool: %v", err)
	}

	// Production order + operation linked to the route operation.
	var poID int64
	if err := pool.QueryRow(ctx, "INSERT INTO production_orders (order_number,item_code,planned_qty,created_by) VALUES ($1,$2,$3,$4) RETURNING id", testutil.UniqueCode(), itemCode, 10, uid).Scan(&poID); err != nil {
		t.Fatalf("seed production_order: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM production_orders WHERE id = $1", poID)
	var pooID int64
	if err := pool.QueryRow(ctx, "INSERT INTO production_order_operations (production_order_id,sequence,operation_name,route_operation_id) VALUES ($1,$2,$3,$4) RETURNING id", poID, 10, "Estampar", routeOp.ID).Scan(&pooID); err != nil {
		t.Fatalf("seed production_order_operation: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM production_order_operations WHERE id = $1", pooID)

	// Complete producing 6 pieces → 6 strokes ≥ 5 limit → alert.
	resp, err := uc.AdvanceOperation(ctx, request.AdvanceOperationDTO{OperationID: pooID, Status: "DONE", ProducedQty: 6})
	if err != nil {
		t.Fatalf("AdvanceOperation: %v", err)
	}
	if len(resp.ToolAlerts) == 0 {
		t.Fatalf("expected a tool replacement alert, got none")
	}

	// The tool's consumed life reflects the produced pieces.
	got, err := tRepo.GetTool(ctx, createdTool.ID)
	if err != nil {
		t.Fatalf("GetTool: %v", err)
	}
	if got.LifeUsed != 6 || !got.NeedsReplacement() {
		t.Errorf("tool life = %v needs=%v, want 6 / true", got.LifeUsed, got.NeedsReplacement())
	}

	// Idempotency: completing an already-DONE operation must NOT consume life again.
	if _, err := uc.AdvanceOperation(ctx, request.AdvanceOperationDTO{OperationID: pooID, Status: "DONE", ProducedQty: 6}); err != nil {
		t.Fatalf("AdvanceOperation (repeat): %v", err)
	}
	got2, err := tRepo.GetTool(ctx, createdTool.ID)
	if err != nil {
		t.Fatalf("GetTool 2: %v", err)
	}
	if got2.LifeUsed != 6 {
		t.Errorf("repeated DONE double-consumed life: %v, want 6", got2.LifeUsed)
	}
}
