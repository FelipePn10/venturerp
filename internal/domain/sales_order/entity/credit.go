package entity

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// CreditDecision is the outcome of evaluating a customer's credit limit against
// their current exposure plus the value of the order being confirmed.
type CreditDecision struct {
	LimitApplies bool            // false when the customer has no credit limit configured
	CreditLimit  decimal.Decimal // configured limit (0 = no limit)
	Exposure     decimal.Decimal // open receivables + other open orders
	OrderValue   decimal.Decimal // value of the order being confirmed
	Available    decimal.Decimal // creditLimit - exposure (may be negative)
	Approved     bool            // whether the order fits within the limit
	Reason       string          // human-readable reason when not approved
}

// EvaluateCredit decides whether confirming an order of orderValue keeps the
// customer within their credit limit. Exposure is the sum of open receivables
// (already-billed, not yet received) and other open orders (confirmed, not yet
// billed). A customer flagged as blocked is never approved. A non-positive
// credit limit means the customer has no limit and is always approved.
//
// A conta é em decimal: em float, 0,10 + 0,20 passa de 0,30 e um pedido que
// fecha exatamente no limite era bloqueado.
func EvaluateCredit(creditLimit, openReceivables, openOrders, orderValue decimal.Decimal, customerBlocked bool) CreditDecision {
	exposure := openReceivables.Add(openOrders)
	d := CreditDecision{
		LimitApplies: creditLimit.IsPositive(),
		CreditLimit:  creditLimit,
		Exposure:     exposure,
		OrderValue:   orderValue,
		Available:    creditLimit.Sub(exposure),
	}

	if customerBlocked {
		d.Approved = false
		d.Reason = "cliente bloqueado para crédito"
		return d
	}

	if !creditLimit.IsPositive() {
		d.Approved = true
		return d
	}

	total := exposure.Add(orderValue)
	if total.GreaterThan(creditLimit) {
		d.Approved = false
		d.Reason = fmt.Sprintf(
			"limite de crédito excedido: limite %s, exposição %s + pedido %s = %s",
			creditLimit.StringFixed(2), exposure.StringFixed(2), orderValue.StringFixed(2), total.StringFixed(2),
		)
		return d
	}

	d.Approved = true
	return d
}
