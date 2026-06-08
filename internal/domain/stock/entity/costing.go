package entity

// CostingState is the weighted-average-cost snapshot of an item's balance in a
// warehouse: on-hand quantity, average unit cost, and total inventory value.
type CostingState struct {
	Quantity  float64
	AvgCost   float64
	TotalCost float64
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
// This is the single source of truth for stock valuation; the repository persists
// whatever this returns. Keeping it pure makes the money math unit-testable.
func ApplyMovementCosting(cur CostingState, delta, unitPrice float64) (next CostingState, lastCost float64) {
	next = CostingState{
		Quantity:  cur.Quantity + delta,
		AvgCost:   cur.AvgCost,
		TotalCost: cur.TotalCost,
	}
	lastCost = cur.AvgCost

	if delta > 0 {
		next.TotalCost = cur.TotalCost + delta*unitPrice
		if next.Quantity > 0 {
			next.AvgCost = next.TotalCost / next.Quantity
		}
		lastCost = unitPrice
		return next, lastCost
	}

	// Outbound: consume at current average.
	next.TotalCost = cur.TotalCost + delta*cur.AvgCost // delta is negative
	if next.TotalCost < 0 || next.Quantity <= 0 {
		next.TotalCost = next.Quantity * next.AvgCost
		if next.TotalCost < 0 {
			next.TotalCost = 0
		}
	}
	return next, lastCost
}
