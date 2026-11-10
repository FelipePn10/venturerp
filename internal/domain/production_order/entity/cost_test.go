package entity

import (
	"github.com/shopspring/decimal"
	"math"
	"testing"

	"github.com/google/uuid"
)

func almostEqual(a decimal.Decimal, b float64) bool { return math.Abs(a.InexactFloat64()-b) < 1e-6 }

func TestBuildSettlement_VariancesAndOverheadApplication(t *testing.T) {
	// Produced 10 units. Real material 1200, real labor 600.
	// Standard per unit: material 100, labor 50, overhead 25.
	in := ActualCostInputs{ProducedQty: decimal.RequireFromString("10"), MaterialCostReal: decimal.RequireFromString("1200"), LaborCostReal: decimal.RequireFromString("600")}
	std := StandardUnitCost{Material: decimal.RequireFromString("100"), Labor: decimal.RequireFromString("50"), Overhead: decimal.RequireFromString("25")}

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

// Centro só com taxa de máquina: o indireto é absorvido sobre a conversão real
// (máquina), na razão do padrão. Antes a base era só mão de obra e dava zero.
func TestBuildSettlement_IndiretoSobreAConversao(t *testing.T) {
	d := decimal.RequireFromString
	in := ActualCostInputs{ProducedQty: d("10"), MaterialCostReal: d("1000"), MachineCostReal: d("400"), SubcontractCostReal: d("150")}
	std := StandardUnitCost{Material: d("90"), Setup: d("5"), Machine: d("25"), Subcontract: d("12"), Overhead: d("15")}

	c := BuildSettlement(1, in, std, "USD", uuid.Nil)

	// 400 × 15 / (5 + 25) = 200.
	if !almostEqual(c.OverheadCostReal, 200) {
		t.Fatalf("indireto real = %v, quer 200", c.OverheadCostReal)
	}
	// Padrão: máquina = (preparação + máquina) × 10 = 300; terceiros 120.
	if !almostEqual(c.MachineCostStd, 300) || !almostEqual(c.SubcontractCostStd, 120) || !almostEqual(c.MaterialCostStd, 900) {
		t.Fatalf("padrão por componente: %+v", c)
	}
	if !almostEqual(c.TotalCostStd, 900+300+120+150) || !almostEqual(c.TotalCostReal, 1000+400+150+200) {
		t.Fatalf("totais: std %v real %v", c.TotalCostStd, c.TotalCostReal)
	}
	if !almostEqual(c.MachineVariance, 100) || !almostEqual(c.SubcontractVariance, 30) || !almostEqual(c.TotalVariance, 280) {
		t.Fatalf("variações: %+v", c)
	}
	if c.Currency != "USD" {
		t.Fatalf("currency = %q, want USD", c.Currency)
	}

	// Padrão sem conversão, mas com indireto: absorve pelo padrão.
	so := BuildSettlement(1, ActualCostInputs{ProducedQty: d("5"), MaterialCostReal: d("500")}, StandardUnitCost{Material: d("90"), Overhead: d("40")}, "", uuid.Nil)
	if !almostEqual(so.OverheadCostReal, 200) {
		t.Fatalf("indireto sem conversão no padrão = %v, quer 200 (40 × 5)", so.OverheadCostReal)
	}
}

func TestBuildSettlement_ZeroProducedQtyNoDivByZero(t *testing.T) {
	in := ActualCostInputs{ProducedQty: decimal.RequireFromString("0"), MaterialCostReal: decimal.RequireFromString("100"), LaborCostReal: decimal.RequireFromString("0")}
	std := StandardUnitCost{Material: decimal.RequireFromString("10"), Labor: decimal.RequireFromString("5"), Overhead: decimal.RequireFromString("2")}

	c := BuildSettlement(1, in, std, "", uuid.Nil)
	if !c.UnitCostReal.IsZero() {
		t.Fatalf("unit real = %v, want 0 for zero produced qty", c.UnitCostReal)
	}
}
