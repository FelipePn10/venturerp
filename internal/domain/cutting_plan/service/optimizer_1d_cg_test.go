package service

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

// TestCG_BeatsBFDOnHeterogeneousStock is the headline proof: on a heterogeneous
// stock set the greedy Best-Fit Decreasing heuristic opens a second full bar and
// leaves a cheaper bar unused, while column generation finds the optimal mix.
//
//	demand : 2×3000, 2×2000   (total 10000)
//	stock  : 6000 ×2, 5000 ×1
//
// BFD packs 3000+3000 into a 6000 bar, then opens a SECOND 6000 bar for the two
// 2000s → 12000 mm consumed, 83.3% utilisation, the 5000 bar wasted.
// Optimal uses one 6000 (3000+3000) and the 5000 (2000+... actually 3000+2000):
// 11000 mm, 90.9%. Column generation must reach 11000.
func TestCG_BeatsBFDOnHeterogeneousStock(t *testing.T) {
	demand := []DemandPiece{
		{PartID: 1, Label: "a", Length: 3000, Qty: 2},
		{PartID: 2, Label: "b", Length: 2000, Qty: 2},
	}
	stock := []StockPiece{
		{StockID: 60, Length: 6000, Qty: 2},
		{StockID: 50, Length: 5000, Qty: 1},
	}

	bfd, err := optimizer1D{}.Optimize(demand, stock, CutParams{})
	if err != nil {
		t.Fatal(err)
	}
	if !almost(bfd.TotalStock, 12000) {
		t.Fatalf("BFD baseline TotalStock = %v, want 12000 (the heuristic wastes a bar)", bfd.TotalStock)
	}

	cg, err := Optimizer(entity.CutTypeLinear1D)
	if err != nil {
		t.Fatal(err)
	}
	sol, err := cg.Optimize(demand, stock, CutParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sol.Unplaced) != 0 {
		t.Fatalf("unplaced = %v, want none", sol.Unplaced)
	}
	if !almost(sol.TotalStock, 11000) {
		t.Fatalf("CG TotalStock = %v, want 11000 (uses the 5000 bar)", sol.TotalStock)
	}
	if sol.TotalStock >= bfd.TotalStock {
		t.Fatalf("CG (%v) did not beat BFD (%v)", sol.TotalStock, bfd.TotalStock)
	}
	if sol.Utilization <= bfd.Utilization {
		t.Fatalf("CG utilisation %.4f did not beat BFD %.4f", sol.Utilization, bfd.Utilization)
	}
	// Every demanded piece must still be present exactly once per unit.
	placed := 0
	for _, pat := range sol.Patterns {
		placed += len(pat.Placements) * max(pat.Repeat, 1)
	}
	if placed != 4 {
		t.Fatalf("placed %d pieces, want 4", placed)
	}
}

// TestCG_UnplacedQuantityIsCorrect guards a regression: when demand exceeds the
// available stock, the residual unplaced pieces must be reported with their full
// quantity (the residual merge once added only one piece per aggregated entry, which
// also tricked the pick-best into preferring an under-counted solution).
func TestCG_UnplacedQuantityIsCorrect(t *testing.T) {
	// 10×2000mm but only one 6000mm bar → 3 placed, 7 unplaced.
	cg, _ := Optimizer(entity.CutTypeLinear1D)
	sol, err := cg.Optimize(
		[]DemandPiece{{PartID: 1, Label: "p", Length: 2000, Qty: 10}},
		[]StockPiece{{StockID: 1, Length: 6000, Qty: 1}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	placed := 0
	for _, pat := range sol.Patterns {
		placed += len(pat.Placements) * max(pat.Repeat, 1)
	}
	if placed != 3 {
		t.Fatalf("placed %d pieces, want 3", placed)
	}
	if got := unplacedQty(sol); got != 7 {
		t.Fatalf("unplaced quantity = %d, want 7", got)
	}
}

// TestCG_ClassicWasteReduction is a second instance: many medium pieces into a
// single fixed bar length where smart pattern selection saves bars over greedy
// best-fit.
//
//	demand : 3×1400, 3×1100  into 3000-long bars (kerf 0)
//
// Greedy: 1400+1400=2800 (waste 200) ×1, then 1400+1100=2500 (waste 500),
// 1100+1100=2200 (waste 800) → 3 bars but uneven. Optimal pairs 1400+1100=2500
// and packs to use exactly 3 bars with the leftover reused — CG must not use MORE
// bars than BFD and must not leave anything unplaced.
func TestCG_NeverWorseThanBFD(t *testing.T) {
	demand := []DemandPiece{
		{PartID: 1, Label: "x", Length: 1400, Qty: 3},
		{PartID: 2, Label: "y", Length: 1100, Qty: 3},
	}
	stock := []StockPiece{{StockID: 30, Length: 3000, Qty: 10}}

	bfd, _ := optimizer1D{}.Optimize(demand, stock, CutParams{})
	cg, _ := Optimizer(entity.CutTypeLinear1D)
	sol, err := cg.Optimize(demand, stock, CutParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sol.Unplaced) != 0 {
		t.Fatalf("unplaced = %v, want none", sol.Unplaced)
	}
	if sol.TotalStock > bfd.TotalStock+1e-6 {
		t.Fatalf("CG consumed more stock (%v) than BFD (%v)", sol.TotalStock, bfd.TotalStock)
	}
	if sol.StockUsed > bfd.StockUsed {
		t.Fatalf("CG used more bars (%d) than BFD (%d)", sol.StockUsed, bfd.StockUsed)
	}
}
