package entity

import (
	"math"
	"testing"

	"github.com/google/uuid"
)

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestBuildSettlement_VariancesAndOverheadApplication(t *testing.T) {
	// Produced 10 units. Real material 1200, real labor 600.
	// Standard per unit: material 100, labor 50, overhead 25.
	in := ActualCostInputs{ProducedQty: 10, MaterialCostReal: 1200, LaborCostReal: 600}
	std := StandardUnitCost{Material: 100, Labor: 50, Overhead: 25}

	c := BuildSettlement(7, in, std, "", uuid.Nil)

	// Overhead applied = labor_real × (overhead_std/labor_std per unit) = 600 × (25/50) = 300.
	if !almostEqual(c.OverheadCostReal, 300) {
		t.Fatalf("overhead real = %v, want 300", c.OverheadCostReal)
	}
	if !almostEqual(c.TotalCostReal, 1200+600+300) {
		t.Fatalf("total real = %v, want 2100", c.TotalCostReal)
	}
	if !almostEqual(c.UnitCostReal, 210) {
		t.Fatalf("unit real = %v, want 210", c.UnitCostReal)
	}

	// Standard totals at produced qty.
	if !almostEqual(c.MaterialCostStd, 1000) || !almostEqual(c.LaborCostStd, 500) || !almostEqual(c.OverheadCostStd, 250) {
		t.Fatalf("std totals wrong: %+v", c)
	}
	if !almostEqual(c.TotalCostStd, 1750) {
		t.Fatalf("total std = %v, want 1750", c.TotalCostStd)
	}

	// Variances (real − std).
	if !almostEqual(c.MaterialVariance, 200) || !almostEqual(c.LaborVariance, 100) ||
		!almostEqual(c.OverheadVariance, 50) || !almostEqual(c.TotalVariance, 350) {
		t.Fatalf("variances wrong: %+v", c)
	}
	if c.Currency != "BRL" {
		t.Fatalf("currency default = %q, want BRL", c.Currency)
	}
}

func TestBuildSettlement_NoStandardLaborSkipsOverhead(t *testing.T) {
	in := ActualCostInputs{ProducedQty: 5, MaterialCostReal: 500, LaborCostReal: 100}
	std := StandardUnitCost{Material: 90, Labor: 0, Overhead: 40}

	c := BuildSettlement(1, in, std, "USD", uuid.Nil)

	if c.OverheadCostReal != 0 {
		t.Fatalf("overhead real = %v, want 0 when std labor is 0", c.OverheadCostReal)
	}
	if !almostEqual(c.TotalCostReal, 600) {
		t.Fatalf("total real = %v, want 600", c.TotalCostReal)
	}
	if c.Currency != "USD" {
		t.Fatalf("currency = %q, want USD", c.Currency)
	}
}

func TestBuildSettlement_ZeroProducedQtyNoDivByZero(t *testing.T) {
	in := ActualCostInputs{ProducedQty: 0, MaterialCostReal: 100, LaborCostReal: 0}
	std := StandardUnitCost{Material: 10, Labor: 5, Overhead: 2}

	c := BuildSettlement(1, in, std, "", uuid.Nil)
	if c.UnitCostReal != 0 {
		t.Fatalf("unit real = %v, want 0 for zero produced qty", c.UnitCostReal)
	}
}
