package service

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

// lTromino is an L-shaped polygon (3 of 4 cells of a 2×2 of side `u`).
func lTromino(u float64) []Point {
	return []Point{{0, 0}, {2 * u, 0}, {2 * u, u}, {u, u}, {u, 2 * u}, {0, 2 * u}}
}

func placementsInBounds(t *testing.T, sol *Solution) {
	t.Helper()
	for _, pat := range sol.Patterns {
		for _, pl := range pat.Placements {
			if pl.X < -eps || pl.Y < -eps || pl.X+pl.W > pat.StockWidth+eps || pl.Y+pl.H > pat.StockHeight+eps {
				t.Fatalf("placement %q out of bounds: x=%.1f y=%.1f w=%.1f h=%.1f on %.0f×%.0f",
					pl.Label, pl.X, pl.Y, pl.W, pl.H, pat.StockWidth, pat.StockHeight)
			}
		}
	}
}

func TestRaster_InterlocksTwoLPiecesOnOneSheet(t *testing.T) {
	o, err := Optimizer(entity.CutTypeTrueShape2D)
	if err != nil {
		t.Fatal(err)
	}
	// Two L-trominoes (bbox 40×40) tile a 40×60 sheet when one is rotated — a
	// bounding-box nester would need two sheets (each bbox is 40×40).
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "L", Polygon: lTromino(20), Qty: 2, AllowRotation: true}},
		[]StockPiece{{StockID: 10, Width: 40, Height: 60, Qty: 2}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(sol.Unplaced) != 0 {
		t.Fatalf("expected both placed, got unplaced %v", sol.Unplaced)
	}
	if sol.StockUsed != 1 {
		t.Fatalf("StockUsed = %d, want 1 (shape-aware interlock); a bbox nester would use 2", sol.StockUsed)
	}
	placementsInBounds(t, sol)
}

func TestRaster_NoRotationNeedsTwoSheets(t *testing.T) {
	o, _ := Optimizer(entity.CutTypeTrueShape2D)
	// Without rotation the two L's cannot interlock → two sheets.
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "L", Polygon: lTromino(20), Qty: 2, AllowRotation: false}},
		[]StockPiece{{StockID: 10, Width: 40, Height: 60, Qty: 2}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sol.StockUsed != 2 {
		t.Fatalf("StockUsed = %d, want 2 (no rotation, no interlock)", sol.StockUsed)
	}
	placementsInBounds(t, sol)
}

func TestRaster_UnplacedWhenTooBig(t *testing.T) {
	o, _ := Optimizer(entity.CutTypeTrueShape2D)
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "big", Polygon: []Point{{0, 0}, {500, 0}, {500, 500}, {0, 500}}, Qty: 1}},
		[]StockPiece{{StockID: 1, Width: 300, Height: 300, Qty: 1}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sol.StockUsed != 0 || len(sol.Unplaced) != 1 {
		t.Fatalf("expected unplaced, got used=%d unplaced=%v", sol.StockUsed, sol.Unplaced)
	}
}
