package service

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

// lNotch is an L-shaped polygon: a square of side a with a b-deep notch cut from its
// top-right corner.
func lNotch(a, b float64) []Point {
	return []Point{{0, 0}, {a, 0}, {a, a - b}, {a - b, a - b}, {a - b, a}, {0, a}}
}

// TestMeta_BeatsGreedyOrderOnIrregularParts is the headline true-shape proof: a mix
// of L-shaped parts where the greedy largest-area order leaves a sheet under-filled,
// while the simulated-annealing search over placement order interlocks them into
// fewer sheets. The metaheuristic must strictly beat the one-shot raster baseline.
func TestMeta_BeatsGreedyOrderOnIrregularParts(t *testing.T) {
	demand := []DemandPiece{
		{PartID: 1, Label: "p", Polygon: lNotch(40, 40.0/3), Qty: 4, AllowRotation: true},
		{PartID: 2, Label: "p", Polygon: lNotch(50, 50.0/3), Qty: 5, AllowRotation: true},
		{PartID: 3, Label: "p", Polygon: lNotch(40, 40.0/3), Qty: 2, AllowRotation: true},
		{PartID: 4, Label: "p", Polygon: lNotch(60, 20), Qty: 4, AllowRotation: true},
		{PartID: 5, Label: "p", Polygon: lNotch(60, 20), Qty: 2, AllowRotation: true},
	}
	stock := []StockPiece{{StockID: 1, Width: 140, Height: 120, Qty: 12}}

	base, err := nestRaster(demand, stock, CutParams{})
	if err != nil {
		t.Fatal(err)
	}
	o, _ := Optimizer(entity.CutTypeTrueShape2D)
	meta, err := o.Optimize(demand, stock, CutParams{})
	if err != nil {
		t.Fatal(err)
	}

	if len(meta.Unplaced) != 0 {
		t.Fatalf("unplaced = %v, want none", meta.Unplaced)
	}
	if meta.StockUsed >= base.StockUsed {
		t.Fatalf("metaheuristic sheets=%d did not beat greedy baseline sheets=%d", meta.StockUsed, base.StockUsed)
	}
	if meta.Utilization <= base.Utilization {
		t.Fatalf("metaheuristic util=%.4f did not beat baseline util=%.4f", meta.Utilization, base.Utilization)
	}
	// All placements must stay within their sheet.
	for _, pat := range meta.Patterns {
		for _, pl := range pat.Placements {
			if pl.X < -eps || pl.Y < -eps || pl.X+pl.W > pat.StockWidth+eps || pl.Y+pl.H > pat.StockHeight+eps {
				t.Fatalf("placement out of bounds: %+v on %.0f×%.0f", pl, pat.StockWidth, pat.StockHeight)
			}
		}
	}
	t.Logf("baseline sheets=%d util=%.4f  ->  metaheuristic sheets=%d util=%.4f",
		base.StockUsed, base.Utilization, meta.StockUsed, meta.Utilization)
}

// TestMeta_NeverWorseThanBaseline guards the safety net: the metaheuristic always
// keeps the greedy order as a candidate, so it can never regress.
func TestMeta_NeverWorseThanBaseline(t *testing.T) {
	demand := []DemandPiece{
		{PartID: 1, Label: "a", Polygon: lNotch(70, 25), Qty: 3, AllowRotation: true},
		{PartID: 2, Label: "b", Polygon: lNotch(45, 15), Qty: 4, AllowRotation: true},
		{PartID: 3, Label: "c", Polygon: []Point{{0, 0}, {30, 0}, {30, 90}, {0, 90}}, Qty: 3, AllowRotation: true},
	}
	stock := []StockPiece{{StockID: 1, Width: 200, Height: 150, Qty: 10}}

	base, _ := nestRaster(demand, stock, CutParams{})
	meta, _ := nestMetaheuristic(demand, stock, CutParams{})
	if unplacedQty(meta) > unplacedQty(base) {
		t.Fatalf("meta left more unplaced (%d) than baseline (%d)", unplacedQty(meta), unplacedQty(base))
	}
	if meta.StockUsed > base.StockUsed {
		t.Fatalf("meta used more sheets (%d) than baseline (%d)", meta.StockUsed, base.StockUsed)
	}
}

// TestMeta_DeterministicReproducible: a fixed RNG seed makes plans reproducible, so
// the same demand+stock always yields the same layout (auditable cutting plans).
func TestMeta_DeterministicReproducible(t *testing.T) {
	demand := []DemandPiece{
		{PartID: 1, Label: "p", Polygon: lNotch(55, 18), Qty: 3, AllowRotation: true},
		{PartID: 2, Label: "q", Polygon: lNotch(35, 12), Qty: 3, AllowRotation: true},
	}
	stock := []StockPiece{{StockID: 1, Width: 160, Height: 130, Qty: 8}}
	a, _ := nestMetaheuristic(demand, stock, CutParams{})
	b, _ := nestMetaheuristic(demand, stock, CutParams{})
	if a.StockUsed != b.StockUsed || a.TotalStock != b.TotalStock || len(a.Patterns) != len(b.Patterns) {
		t.Fatalf("non-deterministic: run A (sheets=%d) != run B (sheets=%d)", a.StockUsed, b.StockUsed)
	}
}
