package service

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

func isCardinal(deg float64) bool {
	return deg == 0 || deg == 90 || deg == 180 || deg == 270
}

// TestFreeRot_DiagonalBarOnlyFitsRotated is the headline FASE 7 proof: a 130×10 bar
// is longer than the 100×100 sheet on every axis-aligned (0/90°) orientation, so a
// 90°-only nester leaves it unplaced. With free rotation the bar nests diagonally
// (~45°, bbox ≈ 99×99) and fits.
func TestFreeRot_DiagonalBarOnlyFitsRotated(t *testing.T) {
	bar := []Point{{0, 0}, {130, 0}, {130, 10}, {0, 10}}
	stock := []StockPiece{{StockID: 1, Width: 100, Height: 100, Qty: 1}}

	o, _ := Optimizer(entity.CutTypeTrueShape2D)

	// With rotation allowed → placed diagonally.
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "bar", Polygon: bar, Qty: 1, AllowRotation: true}},
		stock, CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sol.StockUsed != 1 || len(sol.Unplaced) != 0 {
		t.Fatalf("free rotation should place the bar: used=%d unplaced=%v", sol.StockUsed, sol.Unplaced)
	}
	deg := sol.Patterns[0].Placements[0].RotationDeg
	if isCardinal(deg) {
		t.Fatalf("expected a non-cardinal rotation angle, got %.0f°", deg)
	}

	// Without rotation → cannot fit on any axis-aligned orientation → unplaced.
	norot, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "bar", Polygon: bar, Qty: 1, AllowRotation: false}},
		stock, CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if norot.StockUsed != 0 || len(norot.Unplaced) != 1 {
		t.Fatalf("without rotation the bar must be unplaced: used=%d unplaced=%v", norot.StockUsed, norot.Unplaced)
	}
}
