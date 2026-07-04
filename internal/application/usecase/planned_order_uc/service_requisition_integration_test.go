//go:build integration

package planned_order_uc

import (
	"context"
	"testing"

	"github.com/google/uuid"

	plannedentity "github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	purchasereqRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_requisition"
	routingRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Verifies the subcontracting hook: firming an order whose item has an external
// operation with a service item raises a purchase requisition line for it.
func TestIntegration_FirmGeneratesServiceRequisition(t *testing.T) {
	q, pool := testutil.Queries(t)
	rRepo := routingRepo.New(q)
	reqRepo := purchasereqRepo.New(q, pool)
	ctx := context.Background()
	uid := uuid.New()

	serviceItem := testutil.UniqueCode()
	supplier := testutil.UniqueCode()

	// External operation with subcontracting data.
	op, _ := routingentity.NewOperation(testutil.UniqueCode(), "Zincagem", nil, routingentity.OriginExternal, nil, 0, 0, uid)
	op.ServiceItemCode = &serviceItem
	op.SupplierID = &supplier
	cost := 12.5
	op.CostPerUnit = &cost
	lead := int32(7)
	op.LeadTimeDays = &lead
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
	if _, err := rRepo.AddRouteOperation(ctx, ro); err != nil {
		t.Fatalf("AddRouteOperation: %v", err)
	}

	uc := &FirmPlannedOrderUseCase{ReqRepo: reqRepo, ExternalOps: rRepo, EnterpriseCode: 1}
	order := &plannedentity.PlannedOrder{ItemCode: itemCode, Quantity: 10, CreatedBy: uid}

	reqCode, err := uc.generateServiceRequisition(ctx, order)
	if err != nil {
		t.Fatalf("generateServiceRequisition: %v", err)
	}
	if reqCode == 0 {
		t.Fatal("expected a requisition to be created")
	}
	defer testutil.Exec(t, pool, "DELETE FROM purchase_requisition_items WHERE requisition_code = $1", reqCode)
	defer testutil.Exec(t, pool, "DELETE FROM purchase_requisitions WHERE code = $1", reqCode)

	items, err := reqRepo.ListItems(ctx, reqCode)
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("requisition items = %d, want 1", len(items))
	}
	it := items[0]
	if it.ItemCode != serviceItem || it.Quantity != 10 || it.SuggestedPrice != 12.5 {
		t.Errorf("item = %+v, want service %d qty 10 price 12.5", it, serviceItem)
	}
}
