package service

import (
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
)

func TestApplyLossFormula(t *testing.T) {
	cases := []struct {
		name                         string
		parentQty, childPer, lossPct float64
		formula                      int
		want                         float64
	}{
		{"no loss", 2, 3, 0, 1, 6},
		{"formula1 +10%", 2, 3, 10, 1, 6.6}, // 6 * 1.10
		{"formula2 /(1-10%)", 2, 3, 10, 2, 6.0 / 0.9},
		{"formula3 ignores loss", 2, 3, 10, 3, 6},
		{"default = formula1", 2, 3, 10, 9, 6.6},
		{"formula2 invalid denom falls back", 1, 1, 100, 2, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := applyLossFormula(tc.parentQty, tc.childPer, tc.lossPct, tc.formula)
			if abs(got-tc.want) > 1e-6 {
				t.Errorf("applyLossFormula = %v, want %v", got, tc.want)
			}
		})
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func TestBuildLLCFromBOM(t *testing.T) {
	// 1 → 2 → 3 ; 1 → 3 (3 appears at level 1 and 2 → keep deepest = 2)
	bom := map[int64][]*structentity.ItemStructure{
		1: {{ChildCode: 2}, {ChildCode: 3}},
		2: {{ChildCode: 3}},
	}
	llc := buildLLCFromBOM(bom, []int64{1})
	if llc[1] != 0 {
		t.Errorf("LLC[1] = %d, want 0", llc[1])
	}
	if llc[2] != 1 {
		t.Errorf("LLC[2] = %d, want 1", llc[2])
	}
	if llc[3] != 2 {
		t.Errorf("LLC[3] = %d, want 2 (deepest level)", llc[3])
	}
	if got := maxLLC(llc); got != 2 {
		t.Errorf("maxLLC = %d, want 2", got)
	}
}

func TestExplodeFromBOMWithFormula_MaskFilter(t *testing.T) {
	maskA := "A"
	bom := map[int64][]*structentity.ItemStructure{
		1: {
			{ChildCode: 10, Quantity: 2},                     // generic (no parent mask)
			{ChildCode: 20, Quantity: 1, ParentMask: &maskA}, // only for mask A
		},
	}
	// With mask "A": both apply (generic + A).
	got := explodeFromBOMWithFormula(bom, 1, "A", 5, 0, 1)
	if len(got) != 2 {
		t.Fatalf("expected 2 children for mask A, got %d", len(got))
	}
	// child 10 qty = 5*2 = 10
	if got[0].ItemCode != 10 || got[0].Quantity != 10 {
		t.Errorf("unexpected first child: %+v", got[0])
	}

	// With mask "B": only the generic child applies (A is filtered out).
	gotB := explodeFromBOMWithFormula(bom, 1, "B", 5, 0, 1)
	if len(gotB) != 1 || gotB[0].ItemCode != 10 {
		t.Errorf("expected only generic child for mask B, got %+v", gotB)
	}
}

func TestAggregateInputs(t *testing.T) {
	early := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	inputs := []*entity.MRPInput{
		{ItemCode: 1, Mask: "", Quantity: 5, NeedDate: late},
		{ItemCode: 1, Mask: "", Quantity: 3, NeedDate: early}, // same item+mask → merge, keep earliest
		{ItemCode: 2, Mask: "X", Quantity: 7, NeedDate: late},
	}
	out := aggregateInputs(inputs)
	if len(out) != 2 {
		t.Fatalf("expected 2 aggregated rows, got %d", len(out))
	}
	// sorted by need date → item 1 (early) first.
	if out[0].ItemCode != 1 || out[0].Quantity != 8 {
		t.Errorf("item1 aggregate = %+v, want qty 8", out[0])
	}
	if !out[0].NeedDate.Equal(early) {
		t.Errorf("item1 need date = %v, want earliest %v", out[0].NeedDate, early)
	}
}

func TestFirmSupplyForItem(t *testing.T) {
	need := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	supply := map[int64][]ports.SupplyEntry{
		1: {
			{ItemCode: 1, Quantity: 10, ArrivalDate: need.AddDate(0, 0, -5)}, // before need → counts
			{ItemCode: 1, Quantity: 4, ArrivalDate: need},                    // on need → counts (not after)
			{ItemCode: 1, Quantity: 99, ArrivalDate: need.AddDate(0, 0, 1)},  // after need → excluded
		},
	}
	if got := firmSupplyForItem(supply, 1, need); got != 14 {
		t.Errorf("firmSupplyForItem = %v, want 14", got)
	}
	if got := firmSupplyForItem(supply, 999, need); got != 0 {
		t.Errorf("missing item should yield 0, got %v", got)
	}
}

func TestCriticalPathHours(t *testing.T) {
	// Diamond: 1 → {2,3} → 4. Times: op1=2, op2=3, op3=5, op4=4. No overlap.
	// Path 1-3-4 = 2+5+4 = 11 (critical); 1-2-4 = 2+3+4 = 9.
	ops := []*routingentity.RouteOperation{
		{ID: 1, EffTime: routingentity.OperationTime{Run: 2}},
		{ID: 2, EffTime: routingentity.OperationTime{Run: 3}},
		{ID: 3, EffTime: routingentity.OperationTime{Run: 5}},
		{ID: 4, EffTime: routingentity.OperationTime{Run: 4}},
	}
	edges := []*routingentity.NetworkEdge{
		{PredecessorID: 1, SuccessorID: 2},
		{PredecessorID: 1, SuccessorID: 3},
		{PredecessorID: 2, SuccessorID: 4},
		{PredecessorID: 3, SuccessorID: 4},
	}
	if got := routingentity.CriticalPath(ops, edges, 1).TotalHours; abs(got-11) > 1e-9 {
		t.Errorf("CriticalPath = %v, want 11", got)
	}
}

func TestCriticalPathHours_Overlap(t *testing.T) {
	// Linear 1 → 2 with 50% overlap. op1=10, op2=10.
	// EF(2) = EF(1)*(1-0.5) + op2 = 10*0.5 + 10 = 15.
	ops := []*routingentity.RouteOperation{
		{ID: 1, EffTime: routingentity.OperationTime{Run: 10}},
		{ID: 2, EffTime: routingentity.OperationTime{Run: 10}},
	}
	edges := []*routingentity.NetworkEdge{
		{PredecessorID: 1, SuccessorID: 2, OverlapPct: 50},
	}
	if got := routingentity.CriticalPath(ops, edges, 1).TotalHours; abs(got-15) > 1e-9 {
		t.Errorf("CriticalPath with overlap = %v, want 15", got)
	}
}

func TestCollectAllItemCodes(t *testing.T) {
	bom := map[int64][]*structentity.ItemStructure{
		1: {{ChildCode: 2}, {ChildCode: 3}},
		2: {{ChildCode: 3}},
	}
	codes := collectAllItemCodes(bom, []int64{1})
	set := map[int64]bool{}
	for _, c := range codes {
		set[c] = true
	}
	for _, want := range []int64{1, 2, 3} {
		if !set[want] {
			t.Errorf("missing item code %d in %v", want, codes)
		}
	}
	if len(codes) != 3 {
		t.Errorf("expected 3 distinct codes, got %d (%v)", len(codes), codes)
	}
}
