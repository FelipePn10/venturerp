package service

import (
	"math"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

func almost(a, b float64) bool { return math.Abs(a-b) < 1e-4 }

func opt1D(t *testing.T) CuttingOptimizer {
	t.Helper()
	o, err := Optimizer(entity.CutTypeLinear1D)
	if err != nil {
		t.Fatalf("optimizer for LINEAR_1D not registered: %v", err)
	}
	return o
}

func TestOptimize1D_ExactFitSingleBar(t *testing.T) {
	o := opt1D(t)
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "p", Length: 2000, Qty: 3}},
		[]StockPiece{{StockID: 10, Length: 6000, Qty: 1}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sol.StockUsed != 1 {
		t.Fatalf("StockUsed = %d, want 1", sol.StockUsed)
	}
	if len(sol.Unplaced) != 0 {
		t.Fatalf("unplaced = %v, want none", sol.Unplaced)
	}
	if !almost(sol.Utilization, 1.0) {
		t.Fatalf("utilization = %v, want 1.0", sol.Utilization)
	}
	if len(sol.Patterns) != 1 || len(sol.Patterns[0].Placements) != 3 {
		t.Fatalf("expected 1 pattern with 3 placements, got %+v", sol.Patterns)
	}
	wantOffsets := []float64{0, 2000, 4000}
	for i, pl := range sol.Patterns[0].Placements {
		if !almost(pl.Offset, wantOffsets[i]) {
			t.Errorf("placement %d offset = %v, want %v", i, pl.Offset, wantOffsets[i])
		}
	}
	if !almost(sol.Patterns[0].Remnant, 0) {
		t.Errorf("remnant = %v, want 0", sol.Patterns[0].Remnant)
	}
}

func TestOptimize1D_KerfForcesExtraBar(t *testing.T) {
	o := opt1D(t)
	// With a 5mm kerf, three 2000mm pieces no longer fit in one 6000mm bar:
	// 2000 + 5 + 2000 + 5 + 2000 = 6010 > 6000.
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "p", Length: 2000, Qty: 3}},
		[]StockPiece{{StockID: 10, Length: 6000, Qty: 2}},
		CutParams{Kerf: 5},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sol.StockUsed != 2 {
		t.Fatalf("StockUsed = %d, want 2 (kerf spills to a second bar)", sol.StockUsed)
	}
	if len(sol.Unplaced) != 0 {
		t.Fatalf("unplaced = %v, want none", sol.Unplaced)
	}
}

func TestOptimize1D_BestFitPacksOpenBar(t *testing.T) {
	o := opt1D(t)
	// A 900mm piece should be packed into the slack of the already-open 6000mm
	// bar rather than opening the spare 1000mm stock.
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "big", Length: 2000, Qty: 2}, {PartID: 2, Label: "small", Length: 900, Qty: 1}},
		[]StockPiece{{StockID: 1, Length: 1000, Qty: 1}, {StockID: 2, Length: 6000, Qty: 1}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sol.StockUsed != 1 {
		t.Fatalf("StockUsed = %d, want 1 (small piece packed into open bar)", sol.StockUsed)
	}
	if !almost(sol.TotalDemand, 4900) {
		t.Fatalf("TotalDemand = %v, want 4900", sol.TotalDemand)
	}
}

func TestOptimize1D_RemnantConsumedBeforeFullBar(t *testing.T) {
	o := opt1D(t)
	// A reusable remnant (priority 0) must be opened before a full bar (priority 10).
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "p", Length: 2000, Qty: 1}},
		[]StockPiece{
			{StockID: 99, Length: 6000, Qty: 5, Priority: 10},
			{StockID: 7, Length: 2100, Qty: 1, IsRemnant: true, Priority: 0},
		},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(sol.Patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(sol.Patterns))
	}
	p := sol.Patterns[0]
	if !p.IsRemnant || !almost(p.StockLength, 2100) {
		t.Fatalf("expected the 2100mm remnant to be used, got stock %v isRemnant=%v", p.StockLength, p.IsRemnant)
	}
}

func TestOptimize1D_UnplacedWhenTooLong(t *testing.T) {
	o := opt1D(t)
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "long", Length: 1500, Qty: 1}},
		[]StockPiece{{StockID: 1, Length: 1000, Qty: 2}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sol.StockUsed != 0 || len(sol.Patterns) != 0 {
		t.Fatalf("expected no stock used, got %d", sol.StockUsed)
	}
	if len(sol.Unplaced) != 1 || sol.Unplaced[0].Qty != 1 {
		t.Fatalf("expected 1 unplaced piece, got %+v", sol.Unplaced)
	}
}

func TestOptimize1D_GroupsIdenticalPatterns(t *testing.T) {
	o := opt1D(t)
	// Six 2000mm pieces into 6000mm bars: two identical bars (3 cuts each),
	// grouped into one pattern repeated twice; the third bar stays untouched.
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "p", Length: 2000, Qty: 6}},
		[]StockPiece{{StockID: 10, Length: 6000, Qty: 3}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sol.StockUsed != 2 {
		t.Fatalf("StockUsed = %d, want 2", sol.StockUsed)
	}
	if len(sol.Patterns) != 1 {
		t.Fatalf("expected 1 grouped pattern, got %d", len(sol.Patterns))
	}
	if sol.Patterns[0].Repeat != 2 {
		t.Fatalf("pattern repeat = %d, want 2", sol.Patterns[0].Repeat)
	}
	if !almost(sol.Utilization, 1.0) {
		t.Fatalf("utilization = %v, want 1.0", sol.Utilization)
	}
}

func TestOptimize1D_TrimShiftsOffsetsAndLeavesRemnant(t *testing.T) {
	o := opt1D(t)
	// 100mm head trim → usable 5900; two 2900mm pieces leave a 100mm remnant.
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "p", Length: 2900, Qty: 2}},
		[]StockPiece{{StockID: 10, Length: 6000, Qty: 1}},
		CutParams{Trim: 100},
	)
	if err != nil {
		t.Fatal(err)
	}
	p := sol.Patterns[0]
	wantOffsets := []float64{100, 3000}
	for i, pl := range p.Placements {
		if !almost(pl.Offset, wantOffsets[i]) {
			t.Errorf("placement %d offset = %v, want %v", i, pl.Offset, wantOffsets[i])
		}
	}
	if !almost(p.Remnant, 100) {
		t.Errorf("remnant = %v, want 100", p.Remnant)
	}
}

func TestOptimize1D_RejectsNegativeParams(t *testing.T) {
	o := opt1D(t)
	_, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "p", Length: 100, Qty: 1}},
		[]StockPiece{{StockID: 1, Length: 1000, Qty: 1}},
		CutParams{Kerf: -1},
	)
	if err == nil {
		t.Fatal("expected error for negative kerf")
	}
}
