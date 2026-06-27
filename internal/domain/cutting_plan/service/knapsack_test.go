package service

import (
	"math"
	"testing"
)

func TestBoundedKnapsack_ExactFill(t *testing.T) {
	// Three 2000-long pieces fit a 6000 bar exactly.
	counts, val := boundedKnapsack([]float64{1}, []int{2000}, []int{10}, 6000)
	if counts[0] != 3 {
		t.Fatalf("counts = %v, want [3]", counts)
	}
	if !almost(val, 3) {
		t.Fatalf("value = %v, want 3", val)
	}
}

func TestBoundedKnapsack_RespectsBound(t *testing.T) {
	// Capacity allows 5 but only 2 are demanded.
	counts, val := boundedKnapsack([]float64{1}, []int{1000}, []int{2}, 6000)
	if counts[0] != 2 {
		t.Fatalf("counts = %v, want [2] (bounded by demand)", counts)
	}
	if !almost(val, 2) {
		t.Fatalf("value = %v, want 2", val)
	}
}

func TestBoundedKnapsack_PicksHigherValueMix(t *testing.T) {
	// Two piece types: long (w=2500,v=3) and short (w=1000,v=1). Capacity 6000.
	// Best by value: 2 long (5000, v=6) + 1 short (1000, v=1) = 6000, v=7.
	values := []float64{3, 1}
	weights := []int{2500, 1000}
	bounds := []int{5, 5}
	counts, val := boundedKnapsack(values, weights, bounds, 6000)
	got := counts[0]*weights[0] + counts[1]*weights[1]
	if got > 6000 {
		t.Fatalf("weight %d exceeds capacity", got)
	}
	if !almost(val, 7) {
		t.Fatalf("value = %v, want 7 (counts=%v)", val, counts)
	}
	if counts[0] != 2 || counts[1] != 1 {
		t.Fatalf("counts = %v, want [2 1]", counts)
	}
}

func TestBoundedKnapsack_EmptyWhenNothingFits(t *testing.T) {
	counts, val := boundedKnapsack([]float64{5}, []int{9000}, []int{3}, 6000)
	if counts[0] != 0 || !almost(val, 0) {
		t.Fatalf("expected nothing placed, got counts=%v val=%v", counts, val)
	}
}

func TestBoundedKnapsack_ValueConsistentWithCounts(t *testing.T) {
	values := []float64{2.5, 1.7, 0.9}
	weights := []int{1800, 1200, 700}
	bounds := []int{4, 6, 10}
	counts, val := boundedKnapsack(values, weights, bounds, 6000)
	want := 0.0
	w := 0
	for i := range counts {
		want += float64(counts[i]) * values[i]
		w += counts[i] * weights[i]
	}
	if w > 6000 {
		t.Fatalf("weight %d exceeds capacity", w)
	}
	if math.Abs(val-want) > 1e-9 {
		t.Fatalf("reported value %v != recomputed %v (counts=%v)", val, want, counts)
	}
}
