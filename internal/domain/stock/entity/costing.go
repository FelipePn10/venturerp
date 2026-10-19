package entity

import "github.com/shopspring/decimal"

// CasasCusto é a precisão do custo unitário (avg_cost, unit_price — migração
// 407). Item consumido em milímetro custa frações de centavo por unidade.
const CasasCusto = 10

// CostingState is the weighted-average-cost snapshot of an item's balance in a
// warehouse: on-hand quantity, average unit cost, and total inventory value.
type CostingState struct {
	Quantity  decimal.Decimal
	AvgCost   decimal.Decimal
	TotalCost decimal.Decimal
}

// ApplyMovementCosting returns the balance snapshot after applying a movement of
// `delta` units (signed: positive = inbound, negative = outbound) priced at
// `unitPrice`, along with the resulting "last cost".
//
// Costing rules (weighted average):
//   - Inbound  (delta > 0): the inventory value grows by delta*unitPrice and the
//     average is recomputed as total/quantity. Last cost is the entry unit price.
//   - Outbound (delta < 0): units are consumed at the CURRENT average cost, so the
//     average is unchanged and the value drops by |delta|*avgCost. Guards keep the
//     value from going negative and re-sync it to quantity*avg when stock is fully
//     drained (rounding/over-consumption safety).
//
// Tudo em decimal: em float, 10 entradas de 0,1 não davam 1 e o custo médio
// derivava a cada movimento.
func ApplyMovementCosting(cur CostingState, delta, unitPrice decimal.Decimal) (next CostingState, lastCost decimal.Decimal) {
	next = CostingState{
		Quantity:  cur.Quantity.Add(delta),
		AvgCost:   cur.AvgCost,
		TotalCost: cur.TotalCost,
	}
	lastCost = cur.AvgCost

	if delta.IsPositive() {
		next.TotalCost = cur.TotalCost.Add(delta.Mul(unitPrice))
		if next.Quantity.IsPositive() {
			next.AvgCost = next.TotalCost.DivRound(next.Quantity, CasasCusto)
		}
		lastCost = unitPrice
		return next, lastCost
	}

	// Outbound: consume at current average.
	next.TotalCost = cur.TotalCost.Add(delta.Mul(cur.AvgCost)) // delta is negative
	if next.TotalCost.IsNegative() || !next.Quantity.IsPositive() {
		next.TotalCost = next.Quantity.Mul(next.AvgCost)
		if next.TotalCost.IsNegative() {
			next.TotalCost = decimal.Zero
		}
	}
	return next, lastCost
}
