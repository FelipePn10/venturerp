package service

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

func opt2D(t *testing.T) CuttingOptimizer {
	t.Helper()
	o, err := Optimizer(entity.CutTypeGuillotine2D)
	if err != nil {
		t.Fatalf("optimizer for GUILLOTINE_2D not registered: %v", err)
	}
	return o
}

func TestOptimize2D_ExactTilingSingleSheet(t *testing.T) {
	o := opt2D(t)
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "p", Width: 50, Height: 50, Qty: 4}},
		[]StockPiece{{StockID: 10, Width: 100, Height: 100, Qty: 1}},
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
	if sol.Utilization < 0.999 {
		t.Fatalf("utilization = %v, want ~1.0", sol.Utilization)
	}
	if len(sol.Patterns) != 1 || len(sol.Patterns[0].Placements) != 4 {
		t.Fatalf("expected 1 pattern with 4 placements, got %+v", sol.Patterns)
	}
}

func TestOptimize2D_RotationWhenAllowed(t *testing.T) {
	o := opt2D(t)
	// Part 100×40 doesn't fit a 50×100 sheet unedged, but rotated (40×100) it does.
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "p", Width: 100, Height: 40, Qty: 1, AllowRotation: true}},
		[]StockPiece{{StockID: 10, Width: 50, Height: 100, Qty: 1}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(sol.Unplaced) != 0 || sol.StockUsed != 1 {
		t.Fatalf("expected the part placed via rotation, got unplaced=%v used=%d", sol.Unplaced, sol.StockUsed)
	}
	if !sol.Patterns[0].Placements[0].Rotated {
		t.Fatalf("placement should be marked rotated")
	}
}

func TestOptimize2D_NoRotationLeavesUnplaced(t *testing.T) {
	o := opt2D(t)
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "p", Width: 100, Height: 40, Qty: 1, AllowRotation: false}},
		[]StockPiece{{StockID: 10, Width: 50, Height: 100, Qty: 1}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sol.StockUsed != 0 || len(sol.Unplaced) != 1 {
		t.Fatalf("expected the part unplaced without rotation, got used=%d unplaced=%v", sol.StockUsed, sol.Unplaced)
	}
}

func TestOptimize2D_GrainForbidsRotation(t *testing.T) {
	o := opt2D(t)
	// Rotation allowed but a fixed grain forbids it → cannot fit.
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "p", Width: 100, Height: 40, Qty: 1, AllowRotation: true, Grain: GrainLength}},
		[]StockPiece{{StockID: 10, Width: 50, Height: 100, Qty: 1}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(sol.Unplaced) != 1 {
		t.Fatalf("grain must prevent rotation, expected unplaced, got %+v", sol.Patterns)
	}
}

func TestOptimize2D_OpensSecondSheet(t *testing.T) {
	o := opt2D(t)
	// Two 60×60 parts cannot share a 100×100 sheet (no free rect holds the second).
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "p", Width: 60, Height: 60, Qty: 2}},
		[]StockPiece{{StockID: 10, Width: 100, Height: 100, Qty: 2}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sol.StockUsed != 2 {
		t.Fatalf("StockUsed = %d, want 2", sol.StockUsed)
	}
}

func TestOptimize2D_UnplacedWhenTooBig(t *testing.T) {
	o := opt2D(t)
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "huge", Width: 200, Height: 50, Qty: 1}},
		[]StockPiece{{StockID: 10, Width: 100, Height: 100, Qty: 1}},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sol.StockUsed != 0 || len(sol.Unplaced) != 1 {
		t.Fatalf("expected unplaced, got used=%d unplaced=%v", sol.StockUsed, sol.Unplaced)
	}
}

func TestOptimize2D_RemnantConsumedBeforeFullSheet(t *testing.T) {
	o := opt2D(t)
	sol, err := o.Optimize(
		[]DemandPiece{{PartID: 1, Label: "p", Width: 40, Height: 40, Qty: 1}},
		[]StockPiece{
			{StockID: 99, Width: 2750, Height: 1830, Qty: 5, Priority: 10},
			{StockID: 7, Width: 500, Height: 500, Qty: 1, IsRemnant: true, Priority: 0},
		},
		CutParams{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(sol.Patterns) != 1 || !sol.Patterns[0].IsRemnant {
		t.Fatalf("expected the remnant sheet to be used first, got %+v", sol.Patterns)
	}
}
