package entity

import (
	"math"
	"testing"
)

const eps = 1e-9

func almost(a, b float64) bool { return math.Abs(a-b) < eps }

func TestApplyMovementCosting_FirstInbound(t *testing.T) {
	// Empty balance + buy 10 @ 5.00 → qty 10, avg 5, total 50, last 5.
	next, last := ApplyMovementCosting(CostingState{}, 10, 5.0)
	if !almost(next.Quantity, 10) || !almost(next.AvgCost, 5) || !almost(next.TotalCost, 50) {
		t.Fatalf("got qty=%v avg=%v total=%v", next.Quantity, next.AvgCost, next.TotalCost)
	}
	if !almost(last, 5) {
		t.Fatalf("last cost = %v, want 5", last)
	}
}

func TestApplyMovementCosting_WeightedAverageOnSecondInbound(t *testing.T) {
	// Have 10 @ 5 (total 50). Buy 10 @ 7 (total +70 = 120) → qty 20, avg 6.
	cur := CostingState{Quantity: 10, AvgCost: 5, TotalCost: 50}
	next, last := ApplyMovementCosting(cur, 10, 7.0)
	if !almost(next.Quantity, 20) || !almost(next.AvgCost, 6) || !almost(next.TotalCost, 120) {
		t.Fatalf("got qty=%v avg=%v total=%v, want 20/6/120", next.Quantity, next.AvgCost, next.TotalCost)
	}
	if !almost(last, 7) {
		t.Fatalf("last cost = %v, want 7 (entry price)", last)
	}
}

func TestApplyMovementCosting_OutboundConsumesAtAverage(t *testing.T) {
	// Have 20 @ 6 (total 120). Issue 5 → qty 15, avg unchanged 6, total 90.
	cur := CostingState{Quantity: 20, AvgCost: 6, TotalCost: 120}
	next, last := ApplyMovementCosting(cur, -5, 999) // unitPrice ignored on outbound
	if !almost(next.Quantity, 15) || !almost(next.AvgCost, 6) || !almost(next.TotalCost, 90) {
		t.Fatalf("got qty=%v avg=%v total=%v, want 15/6/90", next.Quantity, next.AvgCost, next.TotalCost)
	}
	if !almost(last, 6) {
		t.Fatalf("last cost = %v, want 6 (avg, unchanged on outbound)", last)
	}
}

func TestApplyMovementCosting_AverageStableAcrossOutbounds(t *testing.T) {
	// The average must not drift when only issuing stock.
	cur := CostingState{Quantity: 100, AvgCost: 3.25, TotalCost: 325}
	for i := 0; i < 7; i++ {
		cur, _ = ApplyMovementCosting(cur, -3, 0)
		if !almost(cur.AvgCost, 3.25) {
			t.Fatalf("avg drifted to %v after issue %d", cur.AvgCost, i)
		}
	}
	if !almost(cur.Quantity, 79) {
		t.Fatalf("qty = %v, want 79", cur.Quantity)
	}
	if !almost(cur.TotalCost, 79*3.25) {
		t.Fatalf("total = %v, want %v", cur.TotalCost, 79*3.25)
	}
}

func TestApplyMovementCosting_FullDrainResetsValueToZero(t *testing.T) {
	// Issue the entire on-hand quantity → qty 0, total re-synced to 0 (no residue).
	cur := CostingState{Quantity: 10, AvgCost: 4, TotalCost: 40}
	next, _ := ApplyMovementCosting(cur, -10, 0)
	if !almost(next.Quantity, 0) {
		t.Fatalf("qty = %v, want 0", next.Quantity)
	}
	if !almost(next.TotalCost, 0) {
		t.Fatalf("total = %v, want 0 after full drain", next.TotalCost)
	}
}

func TestApplyMovementCosting_OverConsumptionGuardsAgainstNegativeValue(t *testing.T) {
	// Consuming more than on-hand must never leave a negative inventory value.
	cur := CostingState{Quantity: 5, AvgCost: 2, TotalCost: 10}
	next, _ := ApplyMovementCosting(cur, -8, 0)
	if next.TotalCost < 0 {
		t.Fatalf("total cost went negative: %v", next.TotalCost)
	}
	if !almost(next.Quantity, -3) {
		t.Fatalf("qty = %v, want -3 (quantity tracked even if value floored)", next.Quantity)
	}
}

func TestApplyMovementCosting_InboundIntoNegativeQuantityKeepsPriorAverage(t *testing.T) {
	// If a prior over-issue left qty negative, a small inbound that does not bring
	// qty positive must not recompute the average (newQty <= 0 guard).
	cur := CostingState{Quantity: -3, AvgCost: 2, TotalCost: 0}
	next, last := ApplyMovementCosting(cur, 2, 9)
	if !almost(next.Quantity, -1) {
		t.Fatalf("qty = %v, want -1", next.Quantity)
	}
	if !almost(next.AvgCost, 2) {
		t.Fatalf("avg = %v, want 2 (unchanged while qty <= 0)", next.AvgCost)
	}
	if !almost(last, 9) {
		t.Fatalf("last = %v, want 9 (entry price on inbound)", last)
	}
}

func TestApplyMovementCosting_ZeroDeltaIsTreatedAsOutboundNoOp(t *testing.T) {
	// delta == 0 takes the non-inbound branch; nothing should change.
	cur := CostingState{Quantity: 12, AvgCost: 8, TotalCost: 96}
	next, last := ApplyMovementCosting(cur, 0, 100)
	if !almost(next.Quantity, 12) || !almost(next.AvgCost, 8) || !almost(next.TotalCost, 96) {
		t.Fatalf("zero delta changed state: %+v", next)
	}
	if !almost(last, 8) {
		t.Fatalf("last = %v, want 8", last)
	}
}
