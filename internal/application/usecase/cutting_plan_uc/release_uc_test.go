package cutting_plan_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	cprepo "github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeCutRepo struct {
	cprepo.CuttingPlanRepository
	plan     *entity.CuttingPlan
	settings *entity.CuttingSettings
	patterns []*entity.CuttingPattern
	stock    []*entity.CuttingStockPiece
	lots     []*entity.LotAvailability
	remnants map[int64]*entity.StockRemnant

	parts      []*entity.CuttingPlanPart
	orderCosts []*entity.CuttingPlanOrderCost

	committedConsumed     []int64
	committedNew          []*entity.StockRemnant
	committedConsumptions []*entity.CuttingPlanConsumption
	released              bool
}

func (f *fakeCutRepo) GetPlanByID(context.Context, int64) (*entity.CuttingPlan, error) {
	return f.plan, nil
}
func (f *fakeCutRepo) GetSettings(context.Context) (*entity.CuttingSettings, error) {
	return f.settings, nil
}
func (f *fakeCutRepo) ListPatterns(context.Context, int64) ([]*entity.CuttingPattern, error) {
	return f.patterns, nil
}
func (f *fakeCutRepo) ListStockPieces(context.Context, int64) ([]*entity.CuttingStockPiece, error) {
	return f.stock, nil
}
func (f *fakeCutRepo) ListParts(context.Context, int64) ([]*entity.CuttingPlanPart, error) {
	return f.parts, nil
}
func (f *fakeCutRepo) ReplaceOrderCosts(_ context.Context, _ int64, costs []*entity.CuttingPlanOrderCost) error {
	f.orderCosts = costs
	return nil
}
func (f *fakeCutRepo) ListAvailableLotsFIFO(context.Context, int64, int64) ([]*entity.LotAvailability, error) {
	return f.lots, nil
}
func (f *fakeCutRepo) GetRemnant(_ context.Context, id int64) (*entity.StockRemnant, error) {
	return f.remnants[id], nil
}
func (f *fakeCutRepo) CommitRelease(_ context.Context, _ int64, consumed []int64, newR []*entity.StockRemnant, cons []*entity.CuttingPlanConsumption) error {
	f.committedConsumed = consumed
	f.committedNew = newR
	f.committedConsumptions = cons
	f.released = true
	return nil
}

type fakeStockRepo struct {
	stockrepo.StockRepository
	balance   *stockentity.StockBalance
	movements []*stockentity.StockMovement
	nextID    int64
}

func (f *fakeStockRepo) GetBalance(context.Context, int64, string, int64) (*stockentity.StockBalance, error) {
	return f.balance, nil
}
func (f *fakeStockRepo) CreateMovement(_ context.Context, m *stockentity.StockMovement) (*stockentity.StockMovement, error) {
	f.nextID++
	m.ID = f.nextID
	f.movements = append(f.movements, m)
	return m, nil
}
func (f *fakeStockRepo) GetLot(context.Context, int64, string) (*stockentity.StockLot, error) {
	return nil, nil
}

func wh(v int64) *int64 { return &v }

func basePlan() *entity.CuttingPlan {
	return &entity.CuttingPlan{
		ID: 1, Code: 100, Status: entity.PlanStatusOptimized,
		MaterialItemCode: 5001, WarehouseID: wh(7), MinRemnantMM: 300,
	}
}

func autoSettings() *entity.CuttingSettings {
	return &entity.CuttingSettings{DefaultConsumptionMode: entity.ConsumptionAutomatic}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestRelease_FullBarAutomatic_PostsBaixa(t *testing.T) {
	cr := &fakeCutRepo{
		plan:     basePlan(),
		settings: autoSettings(),
		patterns: []*entity.CuttingPattern{{StockLengthMM: 6000, RepeatCount: 1, RemnantMM: 0}},
		stock:    []*entity.CuttingStockPiece{{LengthMM: 6000, Quantity: 1}},
	}
	sr := &fakeStockRepo{}
	uc := NewCuttingPlanUseCase(cr, sr, nil)

	res, err := uc.ReleasePlan(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(sr.movements) != 1 || sr.movements[0].MovementType != stockentity.MovementTypeOut || sr.movements[0].Quantity != 1 {
		t.Fatalf("expected one OUT movement of qty 1, got %+v", sr.movements)
	}
	if !cr.released || res.Status != string(entity.PlanStatusReleased) {
		t.Fatalf("plan not firmed: released=%v status=%s", cr.released, res.Status)
	}
	if res.BarsConsumed != 1 || res.RemnantsGenerated != 0 {
		t.Fatalf("bars=%d remnants=%d, want 1/0", res.BarsConsumed, res.RemnantsGenerated)
	}
	if len(cr.committedConsumptions) != 1 || cr.committedConsumptions[0].SourceType != entity.ConsumptionSourceLot {
		t.Fatalf("expected 1 LOT consumption, got %+v", cr.committedConsumptions)
	}
}

func TestRelease_GeneratesReusableRemnant(t *testing.T) {
	cr := &fakeCutRepo{
		plan:     basePlan(),
		settings: autoSettings(),
		patterns: []*entity.CuttingPattern{{StockLengthMM: 6000, RepeatCount: 1, RemnantMM: 500}}, // >= 300
		stock:    []*entity.CuttingStockPiece{{LengthMM: 6000, Quantity: 1}},
	}
	sr := &fakeStockRepo{balance: &stockentity.StockBalance{AvgCost: 12}}
	uc := NewCuttingPlanUseCase(cr, sr, nil)

	res, err := uc.ReleasePlan(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.RemnantsGenerated != 1 || len(cr.committedNew) != 1 {
		t.Fatalf("expected 1 remnant generated, got %d", res.RemnantsGenerated)
	}
	if cr.committedNew[0].LengthMM != 500 {
		t.Fatalf("remnant length = %v, want 500", cr.committedNew[0].LengthMM)
	}
	// Remnant carries the parent's per-UoM cost (size implied by its length).
	if got := cr.committedNew[0].UnitCost; got < 11.99 || got > 12.01 {
		t.Fatalf("remnant unit cost = %v, want 12 (per-UoM)", got)
	}
}

func TestRelease_ConvertsLengthToStockUoM_Meters(t *testing.T) {
	plan := basePlan()
	plan.StockUoM = "M" // material stocked in metres
	cr := &fakeCutRepo{
		plan:     plan,
		settings: autoSettings(),
		patterns: []*entity.CuttingPattern{{StockLengthMM: 6000, RepeatCount: 1, RemnantMM: 0}},
		stock:    []*entity.CuttingStockPiece{{LengthMM: 6000, Quantity: 1}},
	}
	sr := &fakeStockRepo{balance: &stockentity.StockBalance{AvgCost: 10}} // 10 per metre
	uc := NewCuttingPlanUseCase(cr, sr, nil)

	if _, err := uc.ReleasePlan(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if len(sr.movements) != 1 {
		t.Fatalf("expected 1 movement, got %d", len(sr.movements))
	}
	mv := sr.movements[0]
	if mv.Quantity < 5.999 || mv.Quantity > 6.001 {
		t.Fatalf("baixa quantity = %v, want 6 metres", mv.Quantity)
	}
	if mv.TotalPrice < 59.99 || mv.TotalPrice > 60.01 {
		t.Fatalf("baixa total = %v, want 60 (6m × 10)", mv.TotalPrice)
	}
}

func TestRelease_ReusesInventoryRemnant_NoBaixa(t *testing.T) {
	plan := basePlan()
	cr := &fakeCutRepo{
		plan:     plan,
		settings: autoSettings(),
		patterns: []*entity.CuttingPattern{{StockLengthMM: 2000, RepeatCount: 1, IsRemnant: true, RemnantMM: 0}},
		stock:    []*entity.CuttingStockPiece{{LengthMM: 2000, Quantity: 1, IsRemnant: true, RemnantID: wh(50)}},
		remnants: map[int64]*entity.StockRemnant{50: {ID: 50, LengthMM: 2000, UnitCost: 10}},
	}
	sr := &fakeStockRepo{}
	uc := NewCuttingPlanUseCase(cr, sr, nil)

	res, err := uc.ReleasePlan(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(sr.movements) != 0 {
		t.Fatalf("reusing a remnant must not post a stock baixa, got %d movements", len(sr.movements))
	}
	if res.RemnantsConsumed != 1 || len(cr.committedConsumed) != 1 || cr.committedConsumed[0] != 50 {
		t.Fatalf("expected remnant 50 consumed, got %+v", cr.committedConsumed)
	}
	if cr.committedConsumptions[0].SourceType != entity.ConsumptionSourceRemnant {
		t.Fatalf("expected REMNANT consumption source")
	}
}

func TestRelease_2DAreaBaixaAndRemnant(t *testing.T) {
	plan := basePlan()
	plan.CutType = entity.CutTypeGuillotine2D
	plan.StockUoM = "M2" // sheet stocked by square metre
	cr := &fakeCutRepo{
		plan:     plan,
		settings: autoSettings(),
		patterns: []*entity.CuttingPattern{{
			StockWidthMM: 1000, StockHeightMM: 1000, RepeatCount: 1,
			RemnantWidthMM: 400, RemnantHeightMM: 1000, // reusable rectangle (≥ 300 both sides)
		}},
		stock: []*entity.CuttingStockPiece{{WidthMM: 1000, HeightMM: 1000, Quantity: 1}},
	}
	sr := &fakeStockRepo{balance: &stockentity.StockBalance{AvgCost: 5}} // 5 per m²
	uc := NewCuttingPlanUseCase(cr, sr, nil)

	res, err := uc.ReleasePlan(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(sr.movements) != 1 {
		t.Fatalf("expected 1 movement, got %d", len(sr.movements))
	}
	mv := sr.movements[0]
	if mv.Quantity < 0.999 || mv.Quantity > 1.001 { // 1000mm × 1000mm = 1 m²
		t.Fatalf("baixa quantity = %v, want 1 m²", mv.Quantity)
	}
	if mv.TotalPrice < 4.99 || mv.TotalPrice > 5.01 {
		t.Fatalf("baixa total = %v, want 5 (1 m² × 5)", mv.TotalPrice)
	}
	if res.RemnantsGenerated != 1 || len(cr.committedNew) != 1 {
		t.Fatalf("expected 1 2D remnant generated, got %d", res.RemnantsGenerated)
	}
	if cr.committedNew[0].WidthMM != 400 || cr.committedNew[0].HeightMM != 1000 {
		t.Fatalf("2D remnant dims = %v×%v, want 400×1000", cr.committedNew[0].WidthMM, cr.committedNew[0].HeightMM)
	}
}

func TestRelease_AllocatesCostPerOrder(t *testing.T) {
	cr := &fakeCutRepo{
		plan:     basePlan(), // 1D, UoM "" → 1 piece
		settings: autoSettings(),
		patterns: []*entity.CuttingPattern{{StockLengthMM: 6000, RepeatCount: 1}},
		stock:    []*entity.CuttingStockPiece{{LengthMM: 6000, Quantity: 1}},
		parts: []*entity.CuttingPlanPart{
			{LengthMM: 2000, Quantity: 1, SourceRef: strPtr("OP-1")},
			{LengthMM: 4000, Quantity: 1, SourceRef: strPtr("OP-2")},
		},
	}
	sr := &fakeStockRepo{balance: &stockentity.StockBalance{AvgCost: 10}} // 1 bar → total cost 10
	uc := NewCuttingPlanUseCase(cr, sr, nil)

	if _, err := uc.ReleasePlan(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if len(cr.orderCosts) != 2 {
		t.Fatalf("expected 2 order-cost rows, got %d", len(cr.orderCosts))
	}
	byRef := map[string]float64{}
	for _, c := range cr.orderCosts {
		byRef[c.OrderRef] = c.AllocatedCost
	}
	// total cost 10 split 2000:4000 → 3.33 : 6.67
	if got := byRef["OP-1"]; got < 3.32 || got > 3.34 {
		t.Fatalf("OP-1 allocated = %v, want ~3.33", got)
	}
	if got := byRef["OP-2"]; got < 6.66 || got > 6.68 {
		t.Fatalf("OP-2 allocated = %v, want ~6.67", got)
	}
}

func TestAllocateOrderCosts_AddsEdgeBanding(t *testing.T) {
	// Two equal-area parts → material 20 split 10/10. OP-1's part is banded on top and
	// bottom: perimeter = 1000+1000 = 2000 mm = 2 m × R$4/m = R$8 added directly to OP-1.
	parts := []*entity.CuttingPlanPart{
		{WidthMM: 1000, HeightMM: 500, Quantity: 1, SourceRef: strPtr("OP-1"),
			EdgeTop: true, EdgeBottom: true, BandCostPerM: 4},
		{WidthMM: 1000, HeightMM: 500, Quantity: 1, SourceRef: strPtr("OP-2")},
	}
	cons := []*entity.CuttingPlanConsumption{{TotalCost: 20}}
	costs := allocateOrderCosts(parts, cons, true)

	byRef := map[string]float64{}
	for _, c := range costs {
		byRef[c.OrderRef] = c.AllocatedCost
	}
	if got := byRef["OP-1"]; got < 17.99 || got > 18.01 {
		t.Fatalf("OP-1 = %v, want 18 (10 material + 8 banding)", got)
	}
	if got := byRef["OP-2"]; got < 9.99 || got > 10.01 {
		t.Fatalf("OP-2 = %v, want 10 (material only)", got)
	}
}

func TestRelease_RejectsNonOptimizedPlan(t *testing.T) {
	plan := basePlan()
	plan.Status = entity.PlanStatusDraft
	cr := &fakeCutRepo{plan: plan, settings: autoSettings()}
	uc := NewCuttingPlanUseCase(cr, &fakeStockRepo{}, nil)
	if _, err := uc.ReleasePlan(context.Background(), 1); err == nil {
		t.Fatal("expected error firming a non-optimised plan")
	}
}

func TestRelease_ManualModeRequiresLot(t *testing.T) {
	plan := basePlan()
	manual := entity.ConsumptionManual
	plan.LotConsumptionMode = &manual
	cr := &fakeCutRepo{
		plan:     plan,
		settings: autoSettings(),
		patterns: []*entity.CuttingPattern{{StockLengthMM: 6000, RepeatCount: 1}},
		stock:    []*entity.CuttingStockPiece{{LengthMM: 6000, Quantity: 1}}, // no lot
	}
	uc := NewCuttingPlanUseCase(cr, &fakeStockRepo{}, nil)
	if _, err := uc.ReleasePlan(context.Background(), 1); err == nil {
		t.Fatal("expected error: manual mode needs a lot on the stock piece")
	}
}
