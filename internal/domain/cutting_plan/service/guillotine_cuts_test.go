package service

import "testing"

// TestGuillotineCutPlan_TilingTree derives the cut tree for the 70/30/100 tiling of a
// 100×100 sheet: a horizontal cut at y=70 (bottom 100×70, top 100×30), then a vertical
// cut at x=70 splitting the bottom into the 70×70 and 30×70 parts.
func TestGuillotineCutPlan_TilingTree(t *testing.T) {
	boxes := []CutBox{
		{X: 0, Y: 0, W: 70, H: 70, Label: "A"},
		{X: 70, Y: 0, W: 30, H: 70, Label: "B"},
		{X: 0, Y: 70, W: 100, H: 30, Label: "C"},
	}
	cuts := GuillotineCutPlan(boxes, 100, 100)
	if len(cuts) != 2 {
		t.Fatalf("expected 2 cuts, got %d: %+v", len(cuts), cuts)
	}
	// First cut is the primary head cut: horizontal at y=70 spanning the full width.
	if cuts[0].Axis != "HORIZONTAL" || !almost(cuts[0].PositionMM, 70) || cuts[0].Level != 0 {
		t.Fatalf("cut 1 = %+v, want HORIZONTAL @70 level 0", cuts[0])
	}
	if !almost(cuts[0].FromMM, 0) || !almost(cuts[0].ToMM, 100) {
		t.Fatalf("cut 1 span = %v..%v, want 0..100", cuts[0].FromMM, cuts[0].ToMM)
	}
	// Second cut: vertical at x=70 within the bottom panel (level 1).
	if cuts[1].Axis != "VERTICAL" || !almost(cuts[1].PositionMM, 70) || cuts[1].Level != 1 {
		t.Fatalf("cut 2 = %+v, want VERTICAL @70 level 1", cuts[1])
	}
	if !almost(cuts[1].ToMM, 70) {
		t.Fatalf("cut 2 span should be within the bottom 70-high panel, got %v..%v", cuts[1].FromMM, cuts[1].ToMM)
	}
}

// TestGuillotineCutPlan_Grid: a 2×2 grid of equal parts → 1 cut splits it in half, then
// 1 cut per half = 3 cuts total.
func TestGuillotineCutPlan_Grid(t *testing.T) {
	boxes := []CutBox{
		{X: 0, Y: 0, W: 50, H: 50}, {X: 50, Y: 0, W: 50, H: 50},
		{X: 0, Y: 50, W: 50, H: 50}, {X: 50, Y: 50, W: 50, H: 50},
	}
	cuts := GuillotineCutPlan(boxes, 100, 100)
	if len(cuts) != 3 {
		t.Fatalf("expected 3 cuts for a 2×2 grid, got %d", len(cuts))
	}
}

func TestGuillotineCutPlan_SinglePart(t *testing.T) {
	if cuts := GuillotineCutPlan([]CutBox{{X: 0, Y: 0, W: 40, H: 40}}, 100, 100); len(cuts) != 0 {
		t.Fatalf("a single part needs no separating cut, got %d", len(cuts))
	}
}

// TestGuillotineCutPlan_NonGuillotine: a pinwheel layout has no full edge-to-edge cut →
// nil (the caller falls back to placement order).
func TestGuillotineCutPlan_NonGuillotine(t *testing.T) {
	boxes := []CutBox{
		{X: 0, Y: 0, W: 60, H: 40},
		{X: 60, Y: 0, W: 40, H: 60},
		{X: 40, Y: 60, W: 60, H: 40},
		{X: 0, Y: 40, W: 40, H: 60},
	}
	if cuts := GuillotineCutPlan(boxes, 100, 100); cuts != nil {
		t.Fatalf("pinwheel is not guillotine-separable, want nil, got %+v", cuts)
	}
}
