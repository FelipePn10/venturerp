package entity

import (
	"time"

	"github.com/google/uuid"
)

// ProductionOrderCost is the cost settlement of a production order: the actual
// cost incurred on the shop floor, the standard cost it should have had, and the
// variance between them, split into material / labor / overhead.
type ProductionOrderCost struct {
	ID                int64
	ProductionOrderID int64
	ProducedQty       float64

	// Actual cost (total for the order).
	MaterialCostReal float64
	LaborCostReal    float64
	OverheadCostReal float64
	TotalCostReal    float64
	UnitCostReal     float64

	// Standard cost snapshot (unit standard × produced quantity).
	MaterialCostStd float64
	LaborCostStd    float64
	OverheadCostStd float64
	TotalCostStd    float64

	// Variance (actual − standard); positive means it cost more than the standard.
	MaterialVariance float64
	LaborVariance    float64
	OverheadVariance float64
	TotalVariance    float64

	Currency  string
	SettledAt time.Time
	SettledBy uuid.UUID
}

// ActualCostInputs are the raw amounts gathered from the shop floor for an order.
//   - MaterialCostReal: Σ (consumed quantity × weighted-average cost of the item)
//   - LaborCostReal:    Σ (appointment hours × cost/hour of the work center)
type ActualCostInputs struct {
	ProducedQty      float64
	MaterialCostReal float64
	LaborCostReal    float64
}

// StandardUnitCost is the per-unit standard cost split for the produced item.
type StandardUnitCost struct {
	Material float64
	Labor    float64
	Overhead float64
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

	overheadReal := 0.0
	if std.Labor > 0 {
		overheadReal = in.LaborCostReal * (std.Overhead / std.Labor)
	}

	totalReal := in.MaterialCostReal + in.LaborCostReal + overheadReal
	unitReal := 0.0
	if qty > 0 {
		unitReal = totalReal / qty
	}

	materialStd := std.Material * qty
	laborStd := std.Labor * qty
	overheadStd := std.Overhead * qty
	totalStd := materialStd + laborStd + overheadStd

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

		MaterialVariance: in.MaterialCostReal - materialStd,
		LaborVariance:    in.LaborCostReal - laborStd,
		OverheadVariance: overheadReal - overheadStd,
		TotalVariance:    totalReal - totalStd,

		Currency:  currency,
		SettledBy: settledBy,
	}
}
