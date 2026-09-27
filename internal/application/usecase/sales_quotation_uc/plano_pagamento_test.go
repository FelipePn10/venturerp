package sales_quotation_uc

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	quoteentity "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	quoterepo "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository"
)

type repoOrcamentoFake struct {
	quoterepo.SalesQuotationRepository
	orcamento *quoteentity.SalesQuotation
}

func (r *repoOrcamentoFake) GetByCode(context.Context, int64) (*quoteentity.SalesQuotation, error) {
	return r.orcamento, nil
}

type clientesFake struct {
	customerrepo.CustomerRepository
	condicao *customerentity.PaymentCondition
}

func (c *clientesFake) GetPaymentConditionByCode(context.Context, int64) (*customerentity.PaymentCondition, error) {
	return c.condicao, nil
}

func pct(v float64) *float64 { return &v }

// O plano é dividido sobre o que o cliente PAGA. Dividindo só o valor dos
// produtos, cada parcela saía menor que a devida e o plano não fechava com os
// títulos, que o faturamento gera sobre o total do documento fiscal.
func TestPlanoDePagamentoDivideOTotalComImposto(t *testing.T) {
	codigo := int64(7)
	uc := &UseCase{
		Repo: &repoOrcamentoFake{orcamento: &quoteentity.SalesQuotation{
			Code:            1,
			EmissionDate:    time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC),
			PaymentTermCode: &codigo,
			TotalNet:        decimal.NewFromInt(9000), // produtos
			TotalIPI:        decimal.NewFromInt(450),  // IPI
			TotalST:         decimal.NewFromInt(550),  // ICMS-ST
		}},
		Customers: &clientesFake{condicao: &customerentity.PaymentCondition{
			Code: codigo, Description: "50/50",
			Installments: []*customerentity.PaymentInstallment{
				{InstallmentNumber: 1, Percentage: pct(50), DueDays: 28, BaseEvent: string(customerentity.BaseEmissao), IsActive: true},
				{InstallmentNumber: 2, Percentage: pct(50), DueDays: 56, BaseEvent: string(customerentity.BaseEmissao), IsActive: true},
			},
		}},
	}

	plano, err := uc.PlanoDePagamento(context.Background(), 1)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	// 9000 + 450 + 550 = 10.000
	if plano.Total != 10000 {
		t.Fatalf("o total do plano é %.2f; deveria ser 10000 (produto + IPI + ST)", plano.Total)
	}
	if len(plano.Parcelas) != 2 {
		t.Fatalf("esperava 2 parcelas, veio %d", len(plano.Parcelas))
	}
	var soma float64
	for _, p := range plano.Parcelas {
		soma += p.Valor
	}
	if soma != 10000 {
		t.Fatalf("as parcelas somam %.2f e o cliente deve 10000", soma)
	}
	if plano.Parcelas[0].Valor != 5000 {
		t.Fatalf("a primeira parcela é %.2f; com 50%% de 10000 deveria ser 5000", plano.Parcelas[0].Valor)
	}
}

// Orçamento sem imposto continua dividindo o valor dos produtos.
func TestPlanoSemImpostoNaoMuda(t *testing.T) {
	codigo := int64(7)
	uc := &UseCase{
		Repo: &repoOrcamentoFake{orcamento: &quoteentity.SalesQuotation{
			Code: 1, EmissionDate: time.Now(), PaymentTermCode: &codigo,
			TotalNet: decimal.NewFromInt(1000),
		}},
		Customers: &clientesFake{condicao: &customerentity.PaymentCondition{
			Code: codigo, Description: "à vista",
		}},
	}
	plano, err := uc.PlanoDePagamento(context.Background(), 1)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if plano.Total != 1000 {
		t.Fatalf("total %.2f, queria 1000", plano.Total)
	}
}
