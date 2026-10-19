package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

// almost compara decimal com o valor esperado: agora é igualdade exata.
func almost(a decimal.Decimal, b float64) bool { return a.Equal(decimal.NewFromFloat(b)) }

func TestApplyMovementCosting_FirstInbound(t *testing.T) {
	// Empty balance + buy 10 @ 5.00 → qty 10, avg 5, total 50, last 5.
	next, last := ApplyMovementCosting(CostingState{}, d(10), d(5.0))
	if !almost(next.Quantity, 10) || !almost(next.AvgCost, 5) || !almost(next.TotalCost, 50) {
		t.Fatalf("got qty=%v avg=%v total=%v", next.Quantity, next.AvgCost, next.TotalCost)
	}
	if !almost(last, 5) {
		t.Fatalf("last cost = %v, want 5", last)
	}
}

func TestApplyMovementCosting_WeightedAverageOnSecondInbound(t *testing.T) {
	// Have 10 @ 5 (total 50). Buy 10 @ 7 (total +70 = 120) → qty 20, avg 6.
	cur := CostingState{Quantity: d(10), AvgCost: d(5), TotalCost: d(50)}
	next, last := ApplyMovementCosting(cur, d(10), d(7.0))
	if !almost(next.Quantity, 20) || !almost(next.AvgCost, 6) || !almost(next.TotalCost, 120) {
		t.Fatalf("got qty=%v avg=%v total=%v, want 20/6/120", next.Quantity, next.AvgCost, next.TotalCost)
	}
	if !almost(last, 7) {
		t.Fatalf("last cost = %v, want 7 (entry price)", last)
	}
}

func TestApplyMovementCosting_OutboundConsumesAtAverage(t *testing.T) {
	// Have 20 @ 6 (total 120). Issue 5 → qty 15, avg unchanged 6, total 90.
	cur := CostingState{Quantity: d(20), AvgCost: d(6), TotalCost: d(120)}
	next, last := ApplyMovementCosting(cur, d(-5), d(999)) // unitPrice ignored on outbound
	if !almost(next.Quantity, 15) || !almost(next.AvgCost, 6) || !almost(next.TotalCost, 90) {
		t.Fatalf("got qty=%v avg=%v total=%v, want 15/6/90", next.Quantity, next.AvgCost, next.TotalCost)
	}
	if !almost(last, 6) {
		t.Fatalf("last cost = %v, want 6 (avg, unchanged on outbound)", last)
	}
}

func TestApplyMovementCosting_AverageStableAcrossOutbounds(t *testing.T) {
	// The average must not drift when only issuing stock.
	cur := CostingState{Quantity: d(100), AvgCost: d(3.25), TotalCost: d(325)}
	for i := 0; i < 7; i++ {
		cur, _ = ApplyMovementCosting(cur, d(-3), d(0))
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
	cur := CostingState{Quantity: d(10), AvgCost: d(4), TotalCost: d(40)}
	next, _ := ApplyMovementCosting(cur, d(-10), d(0))
	if !almost(next.Quantity, 0) {
		t.Fatalf("qty = %v, want 0", next.Quantity)
	}
	if !almost(next.TotalCost, 0) {
		t.Fatalf("total = %v, want 0 after full drain", next.TotalCost)
	}
}

func TestApplyMovementCosting_OverConsumptionGuardsAgainstNegativeValue(t *testing.T) {
	// Consuming more than on-hand must never leave a negative inventory value.
	cur := CostingState{Quantity: d(5), AvgCost: d(2), TotalCost: d(10)}
	next, _ := ApplyMovementCosting(cur, d(-8), d(0))
	if next.TotalCost.IsNegative() {
		t.Fatalf("total cost went negative: %v", next.TotalCost)
	}
	if !almost(next.Quantity, -3) {
		t.Fatalf("qty = %v, want -3 (quantity tracked even if value floored)", next.Quantity)
	}
}

func TestApplyMovementCosting_InboundIntoNegativeQuantityKeepsPriorAverage(t *testing.T) {
	// If a prior over-issue left qty negative, a small inbound that does not bring
	// qty positive must not recompute the average (newQty <= 0 guard).
	cur := CostingState{Quantity: d(-3), AvgCost: d(2), TotalCost: d(0)}
	next, last := ApplyMovementCosting(cur, d(2), d(9))
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
	cur := CostingState{Quantity: d(12), AvgCost: d(8), TotalCost: d(96)}
	next, last := ApplyMovementCosting(cur, d(0), d(100))
	if !almost(next.Quantity, 12) || !almost(next.AvgCost, 8) || !almost(next.TotalCost, 96) {
		t.Fatalf("zero delta changed state: %+v", next)
	}
	if !almost(last, 8) {
		t.Fatalf("last = %v, want 8", last)
	}
}

// Em decimal, dez entradas de 0,1 somam exatamente 1 — em float davam
// 0,9999999999999999 e o saldo "zerado" nunca zerava.
func TestApplyMovementCosting_DecimalSemDeriva(t *testing.T) {
	cur := CostingState{}
	for i := 0; i < 10; i++ {
		cur, _ = ApplyMovementCosting(cur, decimal.RequireFromString("0.1"), decimal.RequireFromString("3"))
	}
	if !cur.Quantity.Equal(decimal.NewFromInt(1)) || !cur.TotalCost.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("10 × 0,1 deveria dar 1 (R$ 3), deu %s (R$ %s)", cur.Quantity, cur.TotalCost)
	}
	cur, _ = ApplyMovementCosting(cur, decimal.NewFromInt(-1), decimal.Zero)
	if !cur.Quantity.IsZero() || !cur.TotalCost.IsZero() {
		t.Fatalf("saída do saldo inteiro deveria zerar: %s / %s", cur.Quantity, cur.TotalCost)
	}
}

// Custo por milímetro: barra de 6000 mm por R$ 400 → R$ 0,0666666667/mm. Com
// 4 casas virava 0,0667 e 104 mm custavam R$ 6,9368 em vez de R$ 6,9333.
func TestApplyMovementCosting_CustoUnitarioPequeno(t *testing.T) {
	next, _ := ApplyMovementCosting(CostingState{}, decimal.NewFromInt(6000), decimal.RequireFromString("0.0666666667"))
	if next.AvgCost.String() != "0.0666666667" {
		t.Fatalf("custo médio por mm: %s", next.AvgCost)
	}
	saida, _ := ApplyMovementCosting(next, decimal.NewFromInt(-104), decimal.Zero)
	if consumo := next.TotalCost.Sub(saida.TotalCost); consumo.Round(4).String() != "6.9333" {
		t.Fatalf("104 mm deveriam custar 6,9333, custaram %s", consumo)
	}
}
