package production_order_uc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/google/uuid"
)

// ─── fakes ───────────────────────────────────────────────────────────────────

type fakeAuth struct {
	ports.AuthService
	canPlanned bool
	canSales   bool
}

func (f fakeAuth) CanCreatePlannedOrder(context.Context) bool { return f.canPlanned }
func (f fakeAuth) CanUpdateSalesOrder(context.Context) bool   { return f.canSales }

type fakePORepo struct {
	porepo.ProductionOrderRepository
	consumption *poentity.ProductionConsumption
	order       *poentity.ProductionOrder
}

func (f *fakePORepo) AddConsumption(_ context.Context, c *poentity.ProductionConsumption) (*poentity.ProductionConsumption, error) {
	// Echo the input back as if persisted (the use case relies on the saved entity).
	f.consumption = c
	return c, nil
}

func (f *fakePORepo) Complete(context.Context, int64, time.Time) (*poentity.ProductionOrder, error) {
	return f.order, nil
}

type fakeStockRepo struct {
	stockrepo.StockRepository
	movements []*stockentity.StockMovement
	failWith  error
}

func (f *fakeStockRepo) CreateMovement(_ context.Context, m *stockentity.StockMovement) (*stockentity.StockMovement, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	f.movements = append(f.movements, m)
	return m, nil
}

func i64p(v int64) *int64 { return &v }

func TestStructureChildrenForBackflush_SelectsPrimaryAndSkipsOutputs(t *testing.T) {
	raw := []*structentity.ItemStructure{
		{ChildCode: 20, Quantity: 2, SubstituteGroup: 1, SubstitutePriority: 2},
		{ChildCode: 10, Quantity: 3, SubstituteGroup: 1, SubstitutePriority: 1},
		{ChildCode: 30, Quantity: 1, IsCoproduct: true},
		{ChildCode: 40, Quantity: 5, IsFixedQty: true},
	}

	children := structureChildrenForBackflush(raw)

	got := map[int64]structureChild{}
	for _, child := range children {
		got[child.code] = *child
	}
	if _, ok := got[20]; ok {
		t.Error("substituto secundário não deve ser consumido no backflush")
	}
	if _, ok := got[30]; ok {
		t.Error("co-produto não deve ser consumido no backflush")
	}
	if got[10].qty != 3 {
		t.Errorf("primário qty = %v, want 3", got[10].qty)
	}
	if !got[40].fixed {
		t.Error("quantidade fixa deve ser preservada para cálculo por OF")
	}
}

// ─── AddConsumption → OUT movement ──────────────────────────────────────────

func TestAddConsumption_Unauthorized(t *testing.T) {
	uc := AddConsumptionUseCase{Repo: &fakePORepo{}, Auth: fakeAuth{canPlanned: false}}
	_, err := uc.Execute(context.Background(), request.AddConsumptionDTO{})
	if !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestAddConsumption_PostsOutMovementWhenWarehouseSet(t *testing.T) {
	po := &fakePORepo{}
	stock := &fakeStockRepo{}
	uc := AddConsumptionUseCase{Repo: po, Auth: fakeAuth{canPlanned: true}, StockRepo: stock}

	_, err := uc.Execute(context.Background(), request.AddConsumptionDTO{
		ProductionOrderID: 500,
		ItemCode:          1001,
		ConsumedQty:       7,
		WarehouseID:       i64p(3),
		ConsumptionDate:   "2026-04-01",
		CreatedBy:         uuid.New(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stock.movements) != 1 {
		t.Fatalf("expected 1 stock movement, got %d", len(stock.movements))
	}
	m := stock.movements[0]
	if m.MovementType != stockentity.MovementTypeOut {
		t.Fatalf("consumption must post an OUT movement, got %s", m.MovementType)
	}
	if m.ItemCode != 1001 || m.Quantity != 7 || m.WarehouseID != 3 {
		t.Fatalf("movement mapped wrong: %+v", m)
	}
	if m.ReferenceType == nil || *m.ReferenceType != stockentity.ReferenceTypeProductionOrder {
		t.Fatalf("reference type must be PRODUCTION_ORDER")
	}
	if m.ReferenceCode == nil || *m.ReferenceCode != 500 {
		t.Fatalf("reference code must be the production order id")
	}
}

func TestAddConsumption_NoMovementWithoutWarehouse(t *testing.T) {
	stock := &fakeStockRepo{}
	uc := AddConsumptionUseCase{Repo: &fakePORepo{}, Auth: fakeAuth{canPlanned: true}, StockRepo: stock}

	if _, err := uc.Execute(context.Background(), request.AddConsumptionDTO{
		ProductionOrderID: 1, ItemCode: 2, ConsumedQty: 1, ConsumptionDate: "2026-04-01",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stock.movements) != 0 {
		t.Fatal("no warehouse → no stock movement should be posted")
	}
}

func TestAddConsumption_PropagatesMovementError(t *testing.T) {
	stock := &fakeStockRepo{failWith: errors.New("ledger down")}
	uc := AddConsumptionUseCase{Repo: &fakePORepo{}, Auth: fakeAuth{canPlanned: true}, StockRepo: stock}

	_, err := uc.Execute(context.Background(), request.AddConsumptionDTO{
		ProductionOrderID: 1, ItemCode: 2, ConsumedQty: 1, WarehouseID: i64p(9), ConsumptionDate: "2026-04-01",
	})
	if err == nil {
		t.Fatal("stock movement failure should propagate")
	}
}

// ─── CompleteProductionOrder → IN movement ──────────────────────────────────

func TestCompleteProductionOrder_PostsInMovementOfProducedQty(t *testing.T) {
	po := &fakePORepo{order: &poentity.ProductionOrder{
		ID: 88, ItemCode: 2002, Mask: "0001", ProducedQty: 12, PlannedQty: 15, CreatedBy: uuid.New(),
	}}
	stock := &fakeStockRepo{}
	uc := CompleteProductionOrderUseCase{Repo: po, Auth: fakeAuth{canSales: true}, StockRepo: stock}

	_, err := uc.Execute(context.Background(), request.CompleteProductionOrderDTO{
		ID: 88, EndDate: "2026-05-01", WarehouseID: i64p(4),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stock.movements) != 1 {
		t.Fatalf("expected 1 IN movement, got %d", len(stock.movements))
	}
	m := stock.movements[0]
	if m.MovementType != stockentity.MovementTypeIn {
		t.Fatalf("completion must post an IN movement, got %s", m.MovementType)
	}
	if m.ItemCode != 2002 || m.Quantity != 12 || m.WarehouseID != 4 {
		t.Fatalf("finished-goods movement mapped wrong: %+v", m)
	}
}

func TestCompleteProductionOrder_FallsBackToPlannedQty(t *testing.T) {
	// Nothing reported as produced → fall back to the planned quantity.
	po := &fakePORepo{order: &poentity.ProductionOrder{
		ID: 1, ItemCode: 3, ProducedQty: 0, PlannedQty: 20, CreatedBy: uuid.New(),
	}}
	stock := &fakeStockRepo{}
	uc := CompleteProductionOrderUseCase{Repo: po, Auth: fakeAuth{canSales: true}, StockRepo: stock}

	if _, err := uc.Execute(context.Background(), request.CompleteProductionOrderDTO{
		ID: 1, EndDate: "2026-05-01", WarehouseID: i64p(4),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stock.movements) != 1 || stock.movements[0].Quantity != 20 {
		t.Fatalf("expected IN movement of planned qty 20, got %+v", stock.movements)
	}
}

func TestCompleteProductionOrder_NoMovementWhenZeroQty(t *testing.T) {
	po := &fakePORepo{order: &poentity.ProductionOrder{ID: 1, ItemCode: 3, ProducedQty: 0, PlannedQty: 0}}
	stock := &fakeStockRepo{}
	uc := CompleteProductionOrderUseCase{Repo: po, Auth: fakeAuth{canSales: true}, StockRepo: stock}

	if _, err := uc.Execute(context.Background(), request.CompleteProductionOrderDTO{
		ID: 1, EndDate: "2026-05-01", WarehouseID: i64p(4),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stock.movements) != 0 {
		t.Fatal("zero quantity → no finished-goods movement")
	}
}
