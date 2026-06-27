package service

import "testing"

func TestGuillotineKnapsack_ExactTiling(t *testing.T) {
	// Four 50×50 parts (value 1 each) tile a 100×100 plate exactly.
	opts := []gkOption{{item: 0, w: 50, h: 50, rw: 50, rh: 50, value: 1}}
	counts, val, places, _, ok := guillotineKnapsack(opts, 100, 100, 1)
	if !ok {
		t.Fatal("expected ok")
	}
	if counts[0] != 4 || !almost(val, 4) {
		t.Fatalf("counts=%v val=%v, want 4 / 4", counts, val)
	}
	if len(places) != 4 {
		t.Fatalf("places=%d, want 4", len(places))
	}
}

func TestGuillotineKnapsack_ValuePicksBestMix(t *testing.T) {
	// Plate 100×100. Big part 60×100 (value 10) vs small 40×100 (value 3): the
	// guillotine can hold one 60-wide and one 40-wide strip side by side → 13.
	opts := []gkOption{
		{item: 0, w: 60, h: 100, rw: 60, rh: 100, value: 10},
		{item: 1, w: 40, h: 100, rw: 40, rh: 100, value: 3},
	}
	counts, val, _, _, ok := guillotineKnapsack(opts, 100, 100, 2)
	if !ok {
		t.Fatal("expected ok")
	}
	if !almost(val, 13) {
		t.Fatalf("val=%v, want 13 (counts=%v)", val, counts)
	}
	if counts[0] != 1 || counts[1] != 1 {
		t.Fatalf("counts=%v, want [1 1]", counts)
	}
}

func TestGuillotineKnapsack_PositionsWithinPlate(t *testing.T) {
	opts := []gkOption{{item: 0, w: 50, h: 50, rw: 50, rh: 50, value: 1}}
	_, _, places, _, ok := guillotineKnapsack(opts, 100, 100, 1)
	if !ok {
		t.Fatal("expected ok")
	}
	for _, pl := range places {
		if pl.x < 0 || pl.y < 0 || pl.x+50 > 100 || pl.y+50 > 100 {
			t.Fatalf("placement out of plate: %+v", pl)
		}
	}
	// No two 50×50 placements overlap (exact tiling → 4 distinct corners).
	seen := map[[2]int]bool{}
	for _, pl := range places {
		k := [2]int{pl.x, pl.y}
		if seen[k] {
			t.Fatalf("duplicate placement at %v", k)
		}
		seen[k] = true
	}
}

func TestGuillotineKnapsack_BailsOnHugeDiscretisation(t *testing.T) {
	// Many coprime widths on a large plate blow past the discretisation cap.
	var opts []gkOption
	for i := 0; i < 40; i++ {
		w := 101 + 2*i // odd, distinct
		opts = append(opts, gkOption{item: i, w: w, h: 10, rw: float64(w), rh: 10, value: 1})
	}
	_, _, _, _, ok := guillotineKnapsack(opts, 200000, 50, 40)
	if ok {
		t.Fatal("expected ok=false on oversized discretisation")
	}
}
