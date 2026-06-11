package entity

import "fmt"

// CreditDecision is the outcome of evaluating a customer's credit limit against
// their current exposure plus the value of the order being confirmed.
type CreditDecision struct {
	LimitApplies bool    // false when the customer has no credit limit configured
	CreditLimit  float64 // configured limit (0 = no limit)
	Exposure     float64 // open receivables + other open orders
	OrderValue   float64 // value of the order being confirmed
	Available    float64 // creditLimit - exposure (may be negative)
	Approved     bool    // whether the order fits within the limit
	Reason       string  // human-readable reason when not approved
}

// EvaluateCredit decides whether confirming an order of orderValue keeps the
// customer within their credit limit. Exposure is the sum of open receivables
// (already-billed, not yet received) and other open orders (confirmed, not yet
// billed). A customer flagged as blocked is never approved. A non-positive
// credit limit means the customer has no limit and is always approved.
func EvaluateCredit(creditLimit, openReceivables, openOrders, orderValue float64, customerBlocked bool) CreditDecision {
	exposure := openReceivables + openOrders
	d := CreditDecision{
		LimitApplies: creditLimit > 0,
		CreditLimit:  creditLimit,
		Exposure:     exposure,
		OrderValue:   orderValue,
		Available:    creditLimit - exposure,
	}

	if customerBlocked {
		d.Approved = false
		d.Reason = "cliente bloqueado para crédito"
		return d
	}

	if creditLimit <= 0 {
		d.Approved = true
		return d
	}

	if exposure+orderValue > creditLimit {
		d.Approved = false
		d.Reason = fmt.Sprintf(
			"limite de crédito excedido: limite %.2f, exposição %.2f + pedido %.2f = %.2f",
			creditLimit, exposure, orderValue, exposure+orderValue,
		)
		return d
	}

	d.Approved = true
	return d
}
