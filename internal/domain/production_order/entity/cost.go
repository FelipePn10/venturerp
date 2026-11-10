package entity

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

// ProductionOrderCost is the cost settlement of a production order: the actual
// cost incurred on the shop floor, the standard cost it should have had, and the
// variance between them — split into the same components as the item standard
// cost (item_standard_costs): material, machine (setup + machine), labor,
// third-party service and overhead.
type ProductionOrderCost struct {
	ID                int64
	ProductionOrderID int64
	ProducedQty       decimal.Decimal

	// Actual cost (total for the order).
	MaterialCostReal    decimal.Decimal
	MachineCostReal     decimal.Decimal
	LaborCostReal       decimal.Decimal
	SubcontractCostReal decimal.Decimal
	OverheadCostReal    decimal.Decimal
	TotalCostReal       decimal.Decimal
	UnitCostReal        decimal.Decimal

	// Standard cost snapshot (unit standard × produced quantity).
	MaterialCostStd    decimal.Decimal
	MachineCostStd     decimal.Decimal
	LaborCostStd       decimal.Decimal
	SubcontractCostStd decimal.Decimal
	OverheadCostStd    decimal.Decimal
	TotalCostStd       decimal.Decimal

	// Variance (actual − standard); positive means it cost more than the standard.
	MaterialVariance    decimal.Decimal
	MachineVariance     decimal.Decimal
	LaborVariance       decimal.Decimal
	SubcontractVariance decimal.Decimal
	OverheadVariance    decimal.Decimal
	TotalVariance       decimal.Decimal

	Currency  string
	SettledAt time.Time
	SettledBy uuid.UUID
}

// ActualCostInputs are the raw amounts gathered from the shop floor for an order.
//   - MaterialCostReal:    components issued to the order (stock movements) net of returns
//   - MachineCostReal:     hours reported on the order steps × machine rate of the step's work center
//   - LaborCostReal:       hours reported on the order steps × labor rate of the step's work center
//   - SubcontractCostReal: pieces received back from the third party × service price
type ActualCostInputs struct {
	ProducedQty         decimal.Decimal
	MaterialCostReal    decimal.Decimal
	MachineCostReal     decimal.Decimal
	LaborCostReal       decimal.Decimal
	SubcontractCostReal decimal.Decimal
}

// Conversion is the machine + labor actually spent.
func (in ActualCostInputs) Conversion() decimal.Decimal {
	return in.MachineCostReal.Add(in.LaborCostReal)
}

// StandardUnitCost is the per-unit standard cost split for the produced item,
// with the components of item_standard_costs.
type StandardUnitCost struct {
	Material    decimal.Decimal
	Setup       decimal.Decimal
	Machine     decimal.Decimal
	Labor       decimal.Decimal
	Subcontract decimal.Decimal
	Overhead    decimal.Decimal
}

// conversao é a conversão padrão por unidade: preparação + máquina + mão de obra.
func (s StandardUnitCost) conversao() decimal.Decimal {
	return s.Setup.Add(s.Machine).Add(s.Labor)
}

// BuildSettlement combines the shop-floor actuals with the standard cost to
// produce a full cost settlement with variances.
//
// Indireto não é medido no chão de fábrica: é absorvido sobre a conversão real
// (máquina + mão de obra) na razão indireto ÷ conversão do padrão — a mesma base
// que o custo-padrão usou. Sem conversão no padrão, mas com indireto, ele é
// absorvido pelo padrão (indireto unitário × quantidade). Antes a base era só a
// mão de obra: centro com taxa só de máquina zerava o indireto real.
func BuildSettlement(productionOrderID int64, in ActualCostInputs, std StandardUnitCost, currency string, settledBy uuid.UUID) *ProductionOrderCost {
	qty := in.ProducedQty

	materialStd := std.Material.Mul(qty)
	machineStd := std.Setup.Add(std.Machine).Mul(qty)
	laborStd := std.Labor.Mul(qty)
	subStd := std.Subcontract.Mul(qty)
	overheadStd := std.Overhead.Mul(qty)
	totalStd := materialStd.Add(machineStd).Add(laborStd).Add(subStd).Add(overheadStd)

	overheadReal := decimal.Zero
	if conv := std.conversao(); conv.IsPositive() {
		overheadReal = in.Conversion().Mul(std.Overhead).Div(conv)
	} else if std.Overhead.IsPositive() {
		overheadReal = overheadStd
	}

	totalReal := in.MaterialCostReal.Add(in.MachineCostReal).Add(in.LaborCostReal).Add(in.SubcontractCostReal).Add(overheadReal)
	unitReal := decimal.Zero
	if qty.IsPositive() {
		unitReal = totalReal.Div(qty)
	}

	if currency == "" {
		currency = "BRL"
	}

	return &ProductionOrderCost{
		ProductionOrderID: productionOrderID,
		ProducedQty:       qty,

		MaterialCostReal:    in.MaterialCostReal,
		MachineCostReal:     in.MachineCostReal,
		LaborCostReal:       in.LaborCostReal,
		SubcontractCostReal: in.SubcontractCostReal,
		OverheadCostReal:    overheadReal,
		TotalCostReal:       totalReal,
		UnitCostReal:        unitReal,

		MaterialCostStd:    materialStd,
		MachineCostStd:     machineStd,
		LaborCostStd:       laborStd,
		SubcontractCostStd: subStd,
		OverheadCostStd:    overheadStd,
		TotalCostStd:       totalStd,

		MaterialVariance:    in.MaterialCostReal.Sub(materialStd),
		MachineVariance:     in.MachineCostReal.Sub(machineStd),
		LaborVariance:       in.LaborCostReal.Sub(laborStd),
		SubcontractVariance: in.SubcontractCostReal.Sub(subStd),
		OverheadVariance:    overheadReal.Sub(overheadStd),
		TotalVariance:       totalReal.Sub(totalStd),

		Currency:  currency,
		SettledBy: settledBy,
	}
}
