package sales_order_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	custentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
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
	// Lifecycle traz a política e o histórico do grupo econômico; sem ele vale
	// só a validade/classe gravadas no cliente.
	Lifecycle custrepo.CustomerLifecycleRepository
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
	// Na transação da confirmação, trava o cliente antes de somar a exposição:
	// o segundo pedido espera o primeiro gravar e já o enxerga como aberto.
	if t, ok := pedidos.(travaDeCredito); ok {
		if err := t.LockCustomerCredit(ctx, customerCode); err != nil {
			return false, err
		}
	}

	customer, err := c.CustomerRepo.GetCustomerByCode(ctx, customerCode)
	if err != nil {
		return false, fmt.Errorf("lendo o cliente: %w", err)
	}

	// O tipo de análise da condição de pagamento manda antes de tudo.
	if order.PaymentTermCode != nil {
		cond, err := c.CustomerRepo.GetPaymentConditionByCode(ctx, *order.PaymentTermCode)
		if err != nil {
			return false, fmt.Errorf("lendo a condição de pagamento do pedido: %w", err)
		}
		parcelas, err := c.CustomerRepo.ListInstallments(ctx, cond.ID)
		if err != nil {
			return false, fmt.Errorf("lendo as parcelas da condição de pagamento: %w", err)
		}
		analisar, bloquear, motivo := custentity.DecisaoDaCondicao(cond.AnalysisType, custentity.CondicaoAVista(parcelas), customer.Blocked)
		if bloquear {
			if err := pedidos.Block(ctx, code, motivo); err != nil {
				return false, fmt.Errorf("bloqueando o pedido por crédito: %w", err)
			}
			return false, nil
		}
		if !analisar {
			return true, nil
		}
	}

	if ok, reason, err := c.gate(ctx, customer); err != nil {
		return false, err
	} else if !ok {
		if err := pedidos.Block(ctx, code, reason); err != nil {
			return false, fmt.Errorf("bloqueando o pedido por crédito: %w", err)
		}
		return false, nil
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

	decision := entity.EvaluateCredit(customer.CreditLimit, openReceivables, openOrders, orderValue, customer.Blocked)
	if !decision.Approved {
		if err := pedidos.Block(ctx, code, decision.Reason); err != nil {
			return false, fmt.Errorf("bloqueando o pedido por crédito: %w", err)
		}
		return false, nil
	}
	return true, nil
}

// travaDeCredito é implementada pelo repositório ligado à transação.
type travaDeCredito interface {
	LockCustomerCredit(ctx context.Context, customerCode int64) error
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
			// shortcut: uma consulta por pedido aberto; se cliente passar a ter centenas abertos, somar num SQL só (como a Visão 360).
			linhas, err := c.SalesRepo.ListItems(ctx, o.Code)
			if err != nil {
				return decimal.Zero, fmt.Errorf("lendo as linhas do pedido %d: %w", o.Code, err)
			}
			total = total.Add(valorEmAberto(o, linhas))
		}
	}
	return total, nil
}

// valorEmAberto é o que o pedido ainda vai faturar: a parte já faturada virou
// título (entra em openReceivables) e a cancelada não sai mais. O total da
// linha é sobre a quantidade pedida, então vale a fração do saldo. Pedido sem
// linhas lidas fica com o valor da capa.
func valorEmAberto(o *entity.SalesOrder, linhas []*entity.SalesOrderItem) decimal.Decimal {
	if len(linhas) == 0 {
		return orderExposureValue(o)
	}
	total := decimal.Zero
	for _, l := range linhas {
		if l == nil || !l.IsActive || !l.RequestedQty.IsPositive() {
			continue
		}
		saldo := l.RequestedQty.Sub(l.AttendedQty).Sub(l.CancelledQty)
		if !saldo.IsPositive() {
			continue
		}
		valor := l.TotalNetWithIPI.Add(l.TotalST)
		if !valor.IsPositive() {
			valor = l.TotalGross
		}
		total = total.Add(valor.Mul(saldo).Div(l.RequestedQty))
	}
	return total
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
// order header for credit-exposure purposes.
func orderExposureValue(o *entity.SalesOrder) decimal.Decimal {
	switch {
	case o.TotalWithIPIWithST.IsPositive():
		return o.TotalWithIPIWithST
	case o.TotalNet.IsPositive():
		return o.TotalNet
	default:
		return o.TotalGross
	}
}

// gate aplica a análise de crédito do cadastro (custentity.CreditGate).
func (c *CreditChecker) gate(ctx context.Context, customer *custentity.Customer) (bool, string, error) {
	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	if c.Lifecycle == nil {
		ok, reason := custentity.CreditGate(customer, nil, custentity.DefaultCreditPolicy(), today)
		return ok, reason, nil
	}
	pol, err := c.Lifecycle.GetCreditPolicy(ctx)
	if err != nil {
		return false, "", fmt.Errorf("lendo a política de crédito: %w", err)
	}
	group, err := c.Lifecycle.GroupCustomerCodes(ctx, customer.DocumentType, customer.DocumentNumber)
	if err != nil {
		return false, "", fmt.Errorf("lendo o grupo econômico: %w", err)
	}
	found := false
	for _, g := range group {
		found = found || g == customer.Code
	}
	if !found {
		group = append(group, customer.Code)
	}
	hist, err := c.Lifecycle.GetCreditHistory(ctx, group, pol.HistoryMonths, today)
	if err != nil {
		return false, "", fmt.Errorf("lendo o histórico do grupo econômico: %w", err)
	}
	ok, reason := custentity.CreditGate(customer, &hist, pol, today)
	return ok, reason, nil
}
