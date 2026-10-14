package sales_order_uc

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	custrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	finentity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	finrepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

// CreditChecker runs the automatic credit-limit check when an order is
// confirmed. It gathers the customer's exposure (open receivables + other open
// orders) and, if confirming the order would exceed the configured limit (or the
// customer is blocked), blocks the order with a descriptive reason.
type CreditChecker struct {
	SalesRepo     repository.SalesOrderRepository
	CustomerRepo  custrepo.CustomerRepository
	FinancialRepo finrepo.FinancialRepository
}

// Check confere o limite ao confirmar o pedido: true = aprovado, false = o
// pedido foi BLOQUEADO (por `pedidos`, o repositório da transação da
// confirmação). Se a conferência não conseguir rodar, devolve erro e a
// confirmação inteira é desfeita — antes o pedido era aprovado em silêncio, e
// uma falha técnica deixava passar pedido acima do limite.
func (c *CreditChecker) Check(ctx context.Context, code int64, pedidos repository.SalesOrderRepository) (bool, error) {
	order, err := pedidos.GetByCode(ctx, code)
	if err != nil {
		return false, fmt.Errorf("lendo o pedido: %w", err)
	}
	if order.CustomerCode == nil {
		return true, nil
	}
	customerCode := *order.CustomerCode

	customer, err := c.CustomerRepo.GetCustomerByCode(ctx, customerCode)
	if err != nil {
		return false, fmt.Errorf("lendo o cliente: %w", err)
	}

	openReceivables, err := c.openReceivables(ctx, customerCode)
	if err != nil {
		return false, err
	}
	openOrders, err := c.openOrders(ctx, customerCode, code)
	if err != nil {
		return false, err
	}
	orderValue := orderExposureValue(order)

	decision := entity.EvaluateCredit(decimal.NewFromFloat(customer.CreditLimit), openReceivables, openOrders, orderValue, customer.Blocked)
	if !decision.Approved {
		if err := pedidos.Block(ctx, code, decision.Reason); err != nil {
			return false, fmt.Errorf("bloqueando o pedido por crédito: %w", err)
		}
		return false, nil
	}
	return true, nil
}

// openReceivables sums the outstanding balance (billed minus received) of the
// customer's receivables that are still open.
func (c *CreditChecker) openReceivables(ctx context.Context, customerCode int64) (decimal.Decimal, error) {
	cc := customerCode
	list, err := c.FinancialRepo.ListContasReceber(ctx, finrepo.CRFilter{ClienteID: &cc})
	if err != nil {
		return decimal.Zero, fmt.Errorf("lendo os títulos em aberto do cliente: %w", err)
	}
	total := decimal.Zero
	for _, cr := range list {
		if !isOpenReceivable(cr.Status) {
			continue
		}
		total = total.Add(cr.ValorBruto.Sub(cr.ValorRecebido))
	}
	return total, nil
}

// openOrders sums the value of the customer's other confirmed-but-not-yet-billed
// orders, excluding the order being confirmed.
func (c *CreditChecker) openOrders(ctx context.Context, customerCode, excludeCode int64) (decimal.Decimal, error) {
	list, err := c.SalesRepo.ListByCustomer(ctx, customerCode)
	if err != nil {
		return decimal.Zero, fmt.Errorf("lendo os pedidos em aberto do cliente: %w", err)
	}
	total := decimal.Zero
	for _, o := range list {
		if o.Code == excludeCode {
			continue
		}
		switch o.Status {
		case entity.SalesOrderStatusOrder, entity.SalesOrderStatusAnalysis:
			total = total.Add(orderExposureValue(o))
		}
	}
	return total, nil
}

func isOpenReceivable(s finentity.ContaReceberStatus) bool {
	switch s {
	case finentity.ContaReceberStatusPendente,
		finentity.ContaReceberStatusAprovado,
		finentity.ContaReceberStatusVencido,
		finentity.ContaReceberStatusRenegociado:
		return true
	default:
		return false
	}
}

// orderExposureValue picks the most complete monetary value available on the
// order header for credit-exposure purposes. O pedido guarda os totais em
// float; a soma da exposição é feita em decimal a partir daqui.
func orderExposureValue(o *entity.SalesOrder) decimal.Decimal {
	switch {
	case o.TotalWithIPIWithST > 0:
		return decimal.NewFromFloat(o.TotalWithIPIWithST)
	case o.TotalNet > 0:
		return decimal.NewFromFloat(o.TotalNet)
	default:
		return decimal.NewFromFloat(o.TotalGross)
	}
}
