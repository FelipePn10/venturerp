package service

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

// TestCG2D_BeatsHeuristicByGuillotineReasoning is the headline 2D proof. The three
// parts tile a single 100×100 sheet perfectly via guillotine cuts:
//
//	horizontal cut at y=70 → bottom 100×70, top 100×30
//	bottom split vertically → 70×70 + 30×70 ; top holds 100×30
//
// The greedy free-rectangle heuristic places the 70×70 first, fragments the sheet
// into a 30×100 and a 70×30 free rectangle — neither can hold the 100×30 — and opens
// a SECOND sheet. Column generation's guillotine knapsack sees the exact tiling.
func TestCG2D_BeatsHeuristicByGuillotineReasoning(t *testing.T) {
	demand := []DemandPiece{
		{PartID: 1, Label: "A", Width: 70, Height: 70, Qty: 1},
		{PartID: 2, Label: "B", Width: 30, Height: 70, Qty: 1},
		{PartID: 3, Label: "C", Width: 100, Height: 30, Qty: 1},
	}
	stock := []StockPiece{{StockID: 10, Width: 100, Height: 100, Qty: 2}}

	heur, err := optimizer2DGuillotine{}.Optimize(demand, stock, CutParams{})
	if err != nil {
		t.Fatal(err)
	}
	if heur.StockUsed != 2 {
		t.Fatalf("heuristic baseline StockUsed = %d, want 2 (greedy fragments the sheet)", heur.StockUsed)
	}

	o, _ := Optimizer(entity.CutTypeGuillotine2D)
	sol, err := o.Optimize(demand, stock, CutParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sol.Unplaced) != 0 {
		t.Fatalf("unplaced = %v, want none", sol.Unplaced)
	}
	if sol.StockUsed != 1 {
		t.Fatalf("CG StockUsed = %d, want 1 (perfect guillotine tiling)", sol.StockUsed)
	}
	if sol.Utilization < 0.999 {
		t.Fatalf("CG utilisation = %.4f, want ~1.0", sol.Utilization)
	}
	if sol.TotalStock >= heur.TotalStock {
		t.Fatalf("CG (%v) did not beat heuristic (%v)", sol.TotalStock, heur.TotalStock)
	}
}

// TestCG2D_PicksSmallerSheet: a 40×40 part fits both a 100×100 and a 50×50 sheet. The
// heuristic opens the largest sheet (16% utilisation); CG selects the cheaper 50×50.
func TestCG2D_PicksSmallerSheet(t *testing.T) {
	demand := []DemandPiece{{PartID: 1, Label: "p", Width: 40, Height: 40, Qty: 1}}
	stock := []StockPiece{
		{StockID: 1, Width: 100, Height: 100, Qty: 1},
		{StockID: 2, Width: 50, Height: 50, Qty: 1},
	}
	o, _ := Optimizer(entity.CutTypeGuillotine2D)
	sol, err := o.Optimize(demand, stock, CutParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sol.Unplaced) != 0 || sol.StockUsed != 1 {
		t.Fatalf("expected the part placed on one sheet, got used=%d unplaced=%v", sol.StockUsed, sol.Unplaced)
	}
	if !almost(sol.TotalStock, 2500) {
		t.Fatalf("CG TotalStock = %v, want 2500 (the 50×50 sheet)", sol.TotalStock)
	}
}

// TestCG2D_RealisticSizedImprovement scales the guillotine-reasoning win to real
// panel dimensions: three parts tile a 2400×1200 sheet exactly, but the greedy
// heuristic fragments it and opens a second sheet. Column generation packs one.
func TestCG2D_RealisticSizedImprovement(t *testing.T) {
	demand := []DemandPiece{
		{PartID: 1, Label: "A", Width: 1600, Height: 800, Qty: 1},
		{PartID: 2, Label: "B", Width: 800, Height: 800, Qty: 1},
		{PartID: 3, Label: "C", Width: 2400, Height: 400, Qty: 1},
	}
	stock := []StockPiece{{StockID: 1, Width: 2400, Height: 1200, Qty: 2}}

	heur, _ := optimizer2DGuillotine{}.Optimize(demand, stock, CutParams{})
	if heur.StockUsed != 2 {
		t.Fatalf("heuristic baseline StockUsed = %d, want 2", heur.StockUsed)
	}
	o, _ := Optimizer(entity.CutTypeGuillotine2D)
	sol, err := o.Optimize(demand, stock, CutParams{})
	if err != nil {
		t.Fatal(err)
	}
	if sol.StockUsed != 1 || len(sol.Unplaced) != 0 {
		t.Fatalf("CG StockUsed = %d unplaced=%v, want 1 sheet / none", sol.StockUsed, sol.Unplaced)
	}
	if sol.Utilization < 0.999 {
		t.Fatalf("CG utilisation = %.4f, want ~1.0 (perfect tile)", sol.Utilization)
	}
}

// TestCG2D_NeverWorseOnRealisticPanel runs a realistic cabinet job on a standard
// 2750×1830 sheet with kerf/trim and asserts CG places everything and never consumes
// more sheet than the heuristic.
func TestCG2D_NeverWorseOnRealisticPanel(t *testing.T) {
	demand := []DemandPiece{
		{PartID: 1, Label: "lateral", Width: 600, Height: 400, Qty: 8, AllowRotation: true},
		{PartID: 2, Label: "prateleira", Width: 800, Height: 300, Qty: 6, AllowRotation: true},
		{PartID: 3, Label: "fundo", Width: 350, Height: 350, Qty: 10, AllowRotation: true},
		{PartID: 4, Label: "porta", Width: 1200, Height: 500, Qty: 4, AllowRotation: true},
	}
	stock := []StockPiece{{StockID: 1, Width: 2750, Height: 1830, Qty: 20}}
	params := CutParams{Kerf: 4, Trim: 10, MinRemnant: 200}

	heur, _ := optimizer2DGuillotine{}.Optimize(demand, stock, params)
	o, _ := Optimizer(entity.CutTypeGuillotine2D)
	sol, err := o.Optimize(demand, stock, params)
	if err != nil {
		t.Fatal(err)
	}
	if len(sol.Unplaced) != 0 {
		t.Fatalf("unplaced = %v, want none", sol.Unplaced)
	}
	if sol.TotalStock > heur.TotalStock+1e-6 {
		t.Fatalf("CG consumed more sheet (%v) than heuristic (%v)", sol.TotalStock, heur.TotalStock)
	}
	if sol.StockUsed > heur.StockUsed {
		t.Fatalf("CG used more sheets (%d) than heuristic (%d)", sol.StockUsed, heur.StockUsed)
	}
	t.Logf("panel job — heuristic: sheets=%d util=%.4f | CG: sheets=%d util=%.4f",
		heur.StockUsed, heur.Utilization, sol.StockUsed, sol.Utilization)
}
