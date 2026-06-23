package cutting_plan_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	cprepo "github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	plannedrepo "github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository"
	poent "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	prodrepo "github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	strent "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	structqueryrepo "github.com/FelipePn10/panossoerp/internal/domain/structure_query/repository"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeProd struct {
	prodrepo.ProductionOrderRepository
	op *poent.ProductionOrder
}

func (f *fakeProd) GetByCode(context.Context, int64) (*poent.ProductionOrder, error) {
	return f.op, nil
}

type fakeStruct struct {
	structqueryrepo.StructureQueryRepository
	children map[int64][]*strent.ItemStructure
}

func (f *fakeStruct) GetDirectChildrenForMask(_ context.Context, parent int64, _ string) ([]*strent.ItemStructure, error) {
	return f.children[parent], nil
}

type fakeItems struct {
	itemrepo.ItemRepository
	items map[int64]*itementity.Item
}

func (f *fakeItems) FindItemByCode(_ context.Context, ic valueobject.ItemCode) (*itementity.Item, error) {
	if it, ok := f.items[int64(ic)]; ok {
		return it, nil
	}
	return nil, errors.New("item not found")
}

type fakeCut struct {
	cprepo.CuttingPlanRepository
	nextCode int64
	created  []*entity.CuttingPlan
	parts    []*entity.CuttingPlanPart
}

func (f *fakeCut) NextPlanCode(context.Context) (int64, error) { return f.nextCode, nil }
func (f *fakeCut) CreatePlan(_ context.Context, p *entity.CuttingPlan) (*entity.CuttingPlan, error) {
	p.ID = int64(100 + len(f.created))
	f.created = append(f.created, p)
	return p, nil
}
func (f *fakeCut) AddPart(_ context.Context, part *entity.CuttingPlanPart) (*entity.CuttingPlanPart, error) {
	f.parts = append(f.parts, part)
	return part, nil
}

func itemWithDims(code int64, l, w, h int, llc int, uom types.TypeUnitOfMeasurementItem) *itementity.Item {
	it := &itementity.Item{Code: valueobject.ItemCode(code)}
	if l > 0 || w > 0 || h > 0 {
		it.Engineering.Dimensions = &valueobject.Dimensions{Length: l, Width: w, Height: h}
	}
	it.Planning.LLC = llc
	it.Warehouse.UnitOfMeasurement = uom
	return it
}

// ── test ──────────────────────────────────────────────────────────────────────

func TestGenerateFromOrders_1D_AggregatesAndResolvesMaterial(t *testing.T) {
	// Product 1000 → 4× component 2000 (a 720mm leg) → raw material 5000 (a bar).
	prod := &fakeProd{op: &poent.ProductionOrder{OrderNumber: 77, ItemCode: 1000, Mask: "", PlannedQty: 2}}
	str := &fakeStruct{children: map[int64][]*strent.ItemStructure{
		1000: {{ParentCode: 1000, ChildCode: 2000, ChildDescription: "Perna 720", Quantity: 4}},
		2000: {{ParentCode: 2000, ChildCode: 5000, Quantity: 1}}, // component's raw material
	}}
	items := &fakeItems{items: map[int64]*itementity.Item{
		1000: itemWithDims(1000, 0, 0, 0, 1, types.UN),   // finished product (no cut dims)
		2000: itemWithDims(2000, 720, 0, 0, 3, types.UN), // cut piece, 720mm
		5000: itemWithDims(5000, 6000, 0, 0, 9, types.M), // raw bar, LLC 9, stocked in metres
	}}
	cut := &fakeCut{nextCode: 900}

	uc := NewDemandUseCase(cut, items, str, prod, &struct {
		plannedrepo.PlannedOrderRepository
	}{})

	res, err := uc.GenerateFromOrders(context.Background(), request.GenerateCuttingFromOrdersDTO{
		ProductionOrderCodes: []int64{77},
		WarehouseID:          func() *int64 { v := int64(7); return &v }(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Plans) != 1 {
		t.Fatalf("expected 1 plan, got %d (warnings: %v)", len(res.Plans), res.Warnings)
	}
	pl := res.Plans[0]
	if pl.MaterialItemCode != 5000 || pl.CutType != string(entity.CutTypeLinear1D) {
		t.Fatalf("plan material/cuttype = %d/%s, want 5000/LINEAR_1D", pl.MaterialItemCode, pl.CutType)
	}
	if pl.TotalPieces != 8 { // 2 products × 4 legs
		t.Fatalf("total pieces = %d, want 8", pl.TotalPieces)
	}
	if len(cut.parts) != 1 || cut.parts[0].LengthMM != 720 || cut.parts[0].Quantity != 8 {
		t.Fatalf("part wrong: %+v", cut.parts)
	}
	// Single OP fed the material → baixa tied to it; stock UoM snapshotted from item.
	if cut.created[0].ProductionOrderCode == nil || *cut.created[0].ProductionOrderCode != 77 {
		t.Fatalf("production_order_code not set to 77: %+v", cut.created[0].ProductionOrderCode)
	}
	if cut.created[0].StockUoM != types.M {
		t.Fatalf("stock UoM = %v, want M", cut.created[0].StockUoM)
	}
}

func TestGenerateFromOrders_RequiresOrders(t *testing.T) {
	uc := NewDemandUseCase(&fakeCut{}, &fakeItems{}, &fakeStruct{}, &fakeProd{}, &struct {
		plannedrepo.PlannedOrderRepository
	}{})
	if _, err := uc.GenerateFromOrders(context.Background(), request.GenerateCuttingFromOrdersDTO{}); err == nil {
		t.Fatal("expected error when no orders are given")
	}
}
