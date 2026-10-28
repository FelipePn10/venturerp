package entity

import (
	"github.com/shopspring/decimal"
	"time"

	"github.com/google/uuid"
)

// ProductionOrderCost is the cost settlement of a production order: the actual
// cost incurred on the shop floor, the standard cost it should have had, and the
// variance between them, split into material / labor / overhead.
type ProductionOrderCost struct {
	ID                int64
	ProductionOrderID int64
	ProducedQty       decimal.Decimal

	// Actual cost (total for the order).
	MaterialCostReal decimal.Decimal
	LaborCostReal    decimal.Decimal
	OverheadCostReal decimal.Decimal
	TotalCostReal    decimal.Decimal
	UnitCostReal     decimal.Decimal

	// Standard cost snapshot (unit standard × produced quantity).
	MaterialCostStd decimal.Decimal
	LaborCostStd    decimal.Decimal
	OverheadCostStd decimal.Decimal
	TotalCostStd    decimal.Decimal

	// Variance (actual − standard); positive means it cost more than the standard.
	MaterialVariance decimal.Decimal
	LaborVariance    decimal.Decimal
	OverheadVariance decimal.Decimal
	TotalVariance    decimal.Decimal

	Currency  string
	SettledAt time.Time
	SettledBy uuid.UUID
}

// ActualCostInputs are the raw amounts gathered from the shop floor for an order.
//   - MaterialCostReal: Σ (consumed quantity × weighted-average cost of the item)
//   - LaborCostReal:    Σ (appointment hours × cost/hour of the work center)
type ActualCostInputs struct {
	ProducedQty      decimal.Decimal
	MaterialCostReal decimal.Decimal
	LaborCostReal    decimal.Decimal
}

// StandardUnitCost is the per-unit standard cost split for the produced item.
type StandardUnitCost struct {
	Material decimal.Decimal
	Labor    decimal.Decimal
	Overhead decimal.Decimal
}

// BuildSettlement combines the shop-floor actuals with the standard cost to
// produce a full cost settlement with variances.
//
// Overhead is not measured directly on the floor; it is applied on top of the
// real labor using the standard labor:overhead ratio, which keeps the applied
// overhead proportional to the conversion effort actually spent. When the
// standard has no labor reference, no overhead is applied.
func BuildSettlement(productionOrderID int64, in ActualCostInputs, std StandardUnitCost, currency string, settledBy uuid.UUID) *ProductionOrderCost {
	qty := in.ProducedQty

	// Overhead real proporcional à mão de obra real, na razão do padrão.
	overheadReal := decimal.Zero
	if std.Labor.IsPositive() {
		overheadReal = in.LaborCostReal.Mul(std.Overhead).Div(std.Labor)
	}

	totalReal := in.MaterialCostReal.Add(in.LaborCostReal).Add(overheadReal)
	unitReal := decimal.Zero
	if qty.IsPositive() {
		unitReal = totalReal.Div(qty)
	}

	materialStd := std.Material.Mul(qty)
	laborStd := std.Labor.Mul(qty)
	overheadStd := std.Overhead.Mul(qty)
	totalStd := materialStd.Add(laborStd).Add(overheadStd)

	if currency == "" {
		currency = "BRL"
	}

	return &ProductionOrderCost{
		ProductionOrderID: productionOrderID,
		ProducedQty:       qty,

		MaterialCostReal: in.MaterialCostReal,
		LaborCostReal:    in.LaborCostReal,
		OverheadCostReal: overheadReal,
		TotalCostReal:    totalReal,
		UnitCostReal:     unitReal,

		MaterialCostStd: materialStd,
		LaborCostStd:    laborStd,
		OverheadCostStd: overheadStd,
		TotalCostStd:    totalStd,

		MaterialVariance: in.MaterialCostReal.Sub(materialStd),
		LaborVariance:    in.LaborCostReal.Sub(laborStd),
		OverheadVariance: overheadReal.Sub(overheadStd),
		TotalVariance:    totalReal.Sub(totalStd),

		Currency:  currency,
		SettledBy: settledBy,
	}
}
