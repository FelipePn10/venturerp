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
	"github.com/shopspring/decimal"
)

// ─── fakes ───────────────────────────────────────────────────────────────────

type fakeAuth struct {
	ports.AuthService
	canPlanned bool
	canSales   bool
}

func (f fakeAuth) CanCreatePlannedOrder(context.Context) bool { return f.canPlanned }
func (f fakeAuth) CanUpdateSalesOrder(context.Context) bool   { return f.canSales }
func (f fakeAuth) CanGetSalesOrder(context.Context) bool      { return f.canSales }

type fakePORepo struct {
	porepo.ProductionOrderRepository
	consumption      *poentity.ProductionConsumption
	order            *poentity.ProductionOrder
	delivered        decimal.Decimal
	treatExcess      bool
	pendingOCS       bool
	delivery         *poentity.ProductionDelivery
	automatic        map[int64]int64
	deliveries       []*poentity.ProductionDelivery
	appointments     []*poentity.ProductionAppointment
	createdMaterials []*poentity.ProductionOrderMaterial
	hasActivity      bool
	allowQuantity    bool
	allowDates       bool
	acceptsFraction  bool
	materials        []*poentity.ProductionOrderMaterial
	lotAllocations   []poentity.LotAllocation
	scrapDestination *poentity.ScrapDestination
}

func (f *fakePORepo) HasProductionActivity(context.Context, int64) (bool, error) {
	return f.hasActivity, nil
}
func (f *fakePORepo) CanChangeOrderQuantity(context.Context, int64) (bool, error) {
	return f.allowQuantity, nil
}
func (f *fakePORepo) CanChangeOrderDates(context.Context, int64) (bool, error) {
	return f.allowDates, nil
}
func (f *fakePORepo) AcceptsFractionalQuantity(context.Context, int64) (bool, error) {
	return f.acceptsFraction, nil
}
func (f *fakePORepo) Update(_ context.Context, order *poentity.ProductionOrder) (*poentity.ProductionOrder, error) {
	f.order = order
	return order, nil
}
func (f *fakePORepo) ListMaterials(context.Context, int64, poentity.MaterialKind) ([]*poentity.ProductionOrderMaterial, error) {
	return f.materials, nil
}
func (f *fakePORepo) AddMaterial(_ context.Context, material *poentity.ProductionOrderMaterial) (*poentity.ProductionOrderMaterial, error) {
	f.materials = append(f.materials, material)
	return material, nil
}
func (f *fakePORepo) ReplaceMaterial(_ context.Context, _ int64, replacements []poentity.MaterialSubstitution, createdBy uuid.UUID) ([]*poentity.ProductionOrderMaterial, error) {
	result := []*poentity.ProductionOrderMaterial{}
	for _, replacement := range replacements {
		result = append(result, &poentity.ProductionOrderMaterial{ItemCode: replacement.ItemCode, Quantity: replacement.Quantity, CreatedBy: createdBy})
	}
	return result, nil
}
func (f *fakePORepo) DeleteMaterial(context.Context, int64) error { return nil }
func (f *fakePORepo) AllocateLots(_ context.Context, _ int64, _ string, allocations []poentity.LotAllocation, _ uuid.UUID) ([]poentity.LotAllocation, error) {
	f.lotAllocations = allocations
	return allocations, nil
}
func (f *fakePORepo) AllocateLotsBatch(_ context.Context, _ []int64, _ string, lots []poentity.LotAllocation, _ uuid.UUID) ([]poentity.LotAllocation, error) {
	f.lotAllocations = lots
	return lots, nil
}
func (f *fakePORepo) AddScrapDestination(_ context.Context, destination *poentity.ScrapDestination) (*poentity.ScrapDestination, error) {
	f.scrapDestination = destination
	return destination, nil
}
func (f *fakePORepo) UpsertWMSSettings(_ context.Context, settings poentity.WMSWarehouseSettings) (*poentity.WMSWarehouseSettings, error) {
	return &settings, nil
}

func (f *fakePORepo) GetNextOrderNumber(context.Context) (int64, error) { return 100, nil }
func (f *fakePORepo) CreateWithMaterials(_ context.Context, order *poentity.ProductionOrder, materials []*poentity.ProductionOrderMaterial) (*poentity.ProductionOrder, error) {
	f.order = order
	f.createdMaterials = materials
	return order, nil
}

func (f *fakePORepo) ListDeliveries(context.Context, int64) ([]*poentity.ProductionDelivery, error) {
	return f.deliveries, nil
}
func (f *fakePORepo) GetAppointments(context.Context, int64) ([]*poentity.ProductionAppointment, error) {
	return f.appointments, nil
}
func (f *fakePORepo) GetConsumptions(context.Context, int64) ([]*poentity.ProductionConsumption, error) {
	if f.consumption == nil {
		return nil, nil
	}
	return []*poentity.ProductionConsumption{f.consumption}, nil
}

func (f *fakePORepo) GetByCode(context.Context, int64) (*poentity.ProductionOrder, error) {
	return f.order, nil
}
func (f *fakePORepo) GetDeliveredQuantity(context.Context, int64) (decimal.Decimal, error) {
	return f.delivered, nil
}
func (f *fakePORepo) GetDeliveryByIdempotencyKey(context.Context, string) (*poentity.ProductionDelivery, error) {
	return nil, errors.New("not found")
}
func (f *fakePORepo) HasPendingServicePurchaseOrders(context.Context, int64) (bool, error) {
	return f.pendingOCS, nil
}
func (f *fakePORepo) TreatProductionExcess(context.Context) (bool, error) { return f.treatExcess, nil }
func (f *fakePORepo) GetItemAutomaticIssue(_ context.Context, code int64) (bool, int64, error) {
	warehouse, ok := f.automatic[code]
	return ok, warehouse, nil
}
func (f *fakePORepo) RegisterDelivery(_ context.Context, d *poentity.ProductionDelivery) (*poentity.ProductionOrder, error) {
	f.delivery = d
	f.order.Status = poentity.StatusCompleted
	return f.order, nil
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

type fakeDeliveryStructure struct{ children []*structentity.ItemStructure }

func (f *fakeDeliveryStructure) GetAllDirectChildren(context.Context, int64) ([]*structentity.ItemStructure, error) {
	return f.children, nil
}

func TestCreateProductionOrder_GeneratesOnlyDirectPrimaryDemands(t *testing.T) {
	repo := &fakePORepo{automatic: map[int64]int64{20: 2, 30: 3}}
	structure := &fakeDeliveryStructure{children: []*structentity.ItemStructure{
		{ChildCode: 20, Quantity: 2, SubstituteGroup: 1, SubstitutePriority: 1},
		{ChildCode: 21, Quantity: 2, SubstituteGroup: 1, SubstitutePriority: 2},
		{ChildCode: 30, Quantity: 1, IsFixedQty: true},
		{ChildCode: 40, Quantity: 1, IsCoproduct: true},
	}}
	uc := CreateProductionOrderUseCase{Repo: repo, Auth: fakeAuth{canPlanned: true}, Structure: structure}
	order, err := uc.Execute(context.Background(), request.CreateProductionOrderDTO{ItemCode: 10, PlannedQty: 5, CreatedBy: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if order.OrderNumber != 100 || len(repo.createdMaterials) != 2 {
		t.Fatalf("order=%+v materials=%+v", order, repo.createdMaterials)
	}
	if !repo.createdMaterials[0].Quantity.Equal(decimal.NewFromInt(10)) || !repo.createdMaterials[1].Quantity.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("wrong exploded quantities: %+v", repo.createdMaterials)
	}
}

func TestMaintainProductionOrder_QuantityRules(t *testing.T) {
	quantity := decimal.RequireFromString("9.5")
	repo := &fakePORepo{order: &poentity.ProductionOrder{ID: 1, ItemCode: 10, PlannedQty: 10, ProducedQty: 4, Status: poentity.StatusOpen}, allowQuantity: true, acceptsFraction: false}
	uc := MaintainProductionOrderUseCase{Repo: repo, Auth: fakeAuth{canSales: true}}
	if _, err := uc.Execute(context.Background(), request.MaintainProductionOrderDTO{ID: 1, PlannedQty: &quantity}); err == nil {
		t.Fatal("fractional quantity must be rejected")
	}
	repo.acceptsFraction = true
	if _, err := uc.Execute(context.Background(), request.MaintainProductionOrderDTO{ID: 1, PlannedQty: &quantity}); err != nil {
		t.Fatal(err)
	}
	repo.hasActivity = true
	if _, err := uc.Execute(context.Background(), request.MaintainProductionOrderDTO{ID: 1, PlannedQty: &quantity}); err == nil {
		t.Fatal("activity must block maintenance")
	}
}

func TestProductionMaterialControl_AllCommands(t *testing.T) {
	repo := &fakePORepo{}
	uc := ProductionMaterialControlUseCase{Repo: repo, Auth: fakeAuth{canSales: true}}
	ctx := context.Background()
	uid := uuid.New()
	added, err := uc.Add(ctx, request.AddProductionMaterialDTO{ProductionOrderID: 1, Kind: "demand", ItemCode: 2, Quantity: decimal.NewFromInt(3), WarehouseID: 4, CreatedBy: uid})
	if err != nil || added.Kind != poentity.MaterialDemand {
		t.Fatalf("add=%+v err=%v", added, err)
	}
	listed, err := uc.List(ctx, 1, "demand")
	if err != nil || len(listed) != 1 {
		t.Fatalf("list=%+v err=%v", listed, err)
	}
	replaced, err := uc.Replace(ctx, request.ReplaceProductionMaterialDTO{MaterialID: 1, CreatedBy: uid, Replacements: []request.MaterialReplacementDTO{{ItemCode: 3, Quantity: decimal.NewFromInt(2), WarehouseID: 4}}})
	if err != nil || len(replaced) != 1 {
		t.Fatalf("replace=%+v err=%v", replaced, err)
	}
	if err := uc.Delete(ctx, 1); err != nil {
		t.Fatal(err)
	}
	allocations, err := uc.AllocateLots(ctx, request.AllocateProductionLotsDTO{MaterialID: 1, MovementKind: "REQUISITION", CreatedBy: uid, Allocations: []request.LotAllocationDTO{{WarehouseID: 4, Lot: "A", Quantity: decimal.NewFromInt(2)}}})
	if err != nil || len(allocations) != 1 {
		t.Fatalf("allocate=%+v err=%v", allocations, err)
	}
	batch, err := uc.AllocateLotsBatch(ctx, request.BatchAllocateProductionLotsDTO{MaterialIDs: []int64{1, 2}, MovementKind: "REQUISITION", CreatedBy: uid, Lots: []request.LotAllocationDTO{{WarehouseID: 4, Lot: "A", Quantity: decimal.NewFromInt(2)}}})
	if err != nil || len(batch) != 1 {
		t.Fatalf("batch=%+v err=%v", batch, err)
	}
	scrap, err := uc.AddScrap(ctx, request.AddScrapDestinationDTO{ProductionOrderID: 1, ScrapItemCode: 9, WarehouseID: 8, Quantity: decimal.NewFromInt(1), CreatedBy: uid})
	if err != nil || scrap.ScrapItemCode != 9 {
		t.Fatalf("scrap=%+v err=%v", scrap, err)
	}
	intermediate := int64(5)
	if _, err := uc.ConfigureWMS(ctx, request.ConfigureWMSWarehouseDTO{WarehouseID: 4, IsWMS: true, IntermediateOutWarehouseID: &intermediate}); err != nil {
		t.Fatal(err)
	}
}

func (f *fakeStockRepo) CreateMovement(_ context.Context, m *stockentity.StockMovement) (*stockentity.StockMovement, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	f.movements = append(f.movements, m)
	return m, nil
}
func (f *fakeStockRepo) ListMovements(context.Context) ([]*stockentity.StockMovement, error) {
	return f.movements, nil
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
	if m.MovementType != stockentity.MovementTypeProductionEntry {
		t.Fatalf("completion must post an EP movement, got %s", m.MovementType)
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

func TestCompleteProductionOrder_ClassifiesExcessAsEPE(t *testing.T) {
	quantity := decimal.NewFromInt(3)
	po := &fakePORepo{order: &poentity.ProductionOrder{ID: 7, ItemCode: 3, PlannedQty: 10, CreatedBy: uuid.New()}, delivered: decimal.NewFromInt(9), treatExcess: true}
	stock := &fakeStockRepo{}
	uc := CompleteProductionOrderUseCase{Repo: po, Auth: fakeAuth{canSales: true}, StockRepo: stock}
	_, err := uc.Execute(context.Background(), request.CompleteProductionOrderDTO{ID: 7, Quantity: &quantity, WarehouseID: i64p(4), IdempotencyKey: "delivery-7"})
	if err != nil {
		t.Fatal(err)
	}
	if po.delivery == nil || len(po.delivery.Lines) != 2 || len(stock.movements) != 2 {
		t.Fatalf("expected split EPP/EPE delivery, got delivery=%+v movement=%+v", po.delivery, stock.movements)
	}
	if po.delivery.Lines[0].MovementClass != "EPP" || !po.delivery.Lines[0].Quantity.Equal(decimal.NewFromInt(1)) ||
		po.delivery.Lines[1].MovementClass != "EPE" || !po.delivery.Lines[1].Quantity.Equal(decimal.NewFromInt(2)) {
		t.Fatalf("wrong EPP/EPE split: %+v", po.delivery.Lines)
	}
	if stock.movements[0].MovementType != "EPP" || stock.movements[0].Quantity != 1 ||
		stock.movements[1].MovementType != "EPE" || stock.movements[1].Quantity != 2 {
		t.Fatalf("wrong stock movement split: %+v", stock.movements)
	}
}

func TestCompleteProductionOrder_GeneratesREPForAutomaticChildren(t *testing.T) {
	quantity := decimal.NewFromInt(4)
	po := &fakePORepo{order: &poentity.ProductionOrder{ID: 8, ItemCode: 100, PlannedQty: 4, CreatedBy: uuid.New()}, automatic: map[int64]int64{200: 9}}
	stock := &fakeStockRepo{}
	structure := &fakeDeliveryStructure{children: []*structentity.ItemStructure{{ChildCode: 200, Quantity: 2}}}
	uc := CompleteProductionOrderUseCase{Repo: po, Auth: fakeAuth{canSales: true}, StockRepo: stock, Structure: structure}
	_, err := uc.Execute(context.Background(), request.CompleteProductionOrderDTO{ID: 8, Quantity: &quantity, WarehouseID: i64p(4), IdempotencyKey: "delivery-8"})
	if err != nil {
		t.Fatal(err)
	}
	if len(stock.movements) != 2 || stock.movements[1].MovementType != stockentity.MovementTypePlannedRequisition || stock.movements[1].Quantity != 8 || stock.movements[1].Notes == nil || *stock.movements[1].Notes != "REP" {
		t.Fatalf("expected REP movement, got %+v", stock.movements)
	}
}

func TestCompleteProductionOrder_ZeroFinalRejectsPendingOCS(t *testing.T) {
	zero := decimal.Zero
	po := &fakePORepo{order: &poentity.ProductionOrder{ID: 9, PlannedQty: 10}, pendingOCS: true}
	uc := CompleteProductionOrderUseCase{Repo: po, Auth: fakeAuth{canSales: true}}
	_, err := uc.Execute(context.Background(), request.CompleteProductionOrderDTO{ID: 9, Quantity: &zero, Final: true, WarehouseID: i64p(4), IdempotencyKey: "delivery-9"})
	if err == nil {
		t.Fatal("expected pending OCS guard")
	}
}

func TestOperationalConsultation_AggregatesExecution(t *testing.T) {
	refType := stockentity.ReferenceTypeProductionOrder
	refCode := int64(55)
	po := &fakePORepo{order: &poentity.ProductionOrder{ID: 55, ItemCode: 100, PlannedQty: 10, ProducedQty: 7},
		deliveries: []*poentity.ProductionDelivery{{ID: 1, ProductionOrderID: 55, Quantity: decimal.NewFromInt(6), MovementClass: "EPP", WarehouseID: 4}}}
	stock := &fakeStockRepo{movements: []*stockentity.StockMovement{
		{ID: 1, ItemCode: 100, MovementType: "EPP", Quantity: 6, ReferenceType: &refType, ReferenceCode: &refCode},
		{ID: 2, ItemCode: 999, MovementType: "IN", Quantity: 1},
	}}
	uc := OperationalConsultationUseCase{Repo: po, Stock: stock, Auth: fakeAuth{canSales: true}}
	result, err := uc.Execute(context.Background(), 55)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Deliveries) != 1 || len(result.Movements) != 1 || !result.Totals["delivered"].Equal(decimal.NewFromInt(6)) || !result.Totals["pending"].Equal(decimal.NewFromInt(4)) {
		t.Fatalf("unexpected operational result: %+v", result)
	}
}
