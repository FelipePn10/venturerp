package service

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

func TestPolygonBBox(t *testing.T) {
	// An L-shaped polygon spanning 0..600 in x and 0..400 in y.
	poly := []Point{{0, 0}, {600, 0}, {600, 200}, {300, 200}, {300, 400}, {0, 400}}
	w, h := PolygonBBox(poly)
	if w != 600 || h != 400 {
		t.Fatalf("bbox = %v×%v, want 600×400", w, h)
	}
}

func TestTrueShape_NestsViaBoundingBox(t *testing.T) {
	o, err := Optimizer(entity.CutTypeTrueShape2D)
	if err != nil {
		t.Fatalf("true-shape optimizer not registered: %v", err)
	}
	// Two L-shaped parts (bbox 500×500) onto one 1000×1000 sheet.
	poly := []Point{{0, 0}, {500, 0}, {500, 250}, {250, 250}, {250, 500}, {0, 500}}
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "L", Polygon: poly, Qty: 2}},
		[]StockPiece{{StockID: 10, Width: 1000, Height: 1000, Qty: 1}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sol.StockUsed != 1 || len(sol.Unplaced) != 0 {
		t.Fatalf("expected both parts on one sheet, got used=%d unplaced=%v", sol.StockUsed, sol.Unplaced)
	}
	total := 0
	for _, pat := range sol.Patterns {
		total += len(pat.Placements) * pat.Repeat
	}
	if total != 2 {
		t.Fatalf("expected 2 placements, got %d", total)
	}
}

func TestTrueShape_DerivesBBoxAndPlaces(t *testing.T) {
	o, _ := Optimizer(entity.CutTypeTrueShape2D)
	// A polygon whose bbox (700×400) is too tall to fit a 700×300 sheet → unplaced.
	poly := []Point{{0, 0}, {700, 0}, {700, 400}, {0, 400}}
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "P", Polygon: poly, Qty: 1}},
		[]StockPiece{{StockID: 1, Width: 700, Height: 300, Qty: 1}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sol.StockUsed != 0 || len(sol.Unplaced) != 1 {
		t.Fatalf("expected the part unplaced, got used=%d unplaced=%v", sol.StockUsed, sol.Unplaced)
	}
}
