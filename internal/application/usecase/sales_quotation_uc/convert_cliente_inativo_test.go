package sales_quotation_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	quoteentity "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
)

type clienteSituacaoFake struct {
	customerrepo.CustomerRepository
	ativo bool
}

func (c clienteSituacaoFake) GetCustomerByCode(_ context.Context, code int64) (*customerentity.Customer, error) {
	return &customerentity.Customer{Code: code, IsActive: c.ativo}, nil
}

// Orçamento feito com o cliente ativo não vira pedido depois que ele é inativado.
func TestConversaoRecusaClienteInativo(t *testing.T) {
	cliente := int64(5)
	q := &quoteentity.SalesQuotation{Code: 1, CustomerCode: &cliente, Status: quoteentity.SalesQuotationStatusDraft}
	itens := []*quoteentity.SalesQuotationItem{{Code: 10, ItemCode: 99, RequestedQty: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100),
		IsActive: true, Status: quoteentity.SalesQuotationItemStatusOpen}}
	pedidos := &pedidosComStatus{}
	uc := &ConvertUseCase{Quotes: &UseCase{Repo: conversaoRepo{q: q, itens: itens}, Auth: quotationRepresentativeAuth{}, Tributos: sugestorConversao{},
		Customers: clienteSituacaoFake{ativo: false}}, UOW: uowPedidos{pedidos}, Confirmar: &confirmadorFake{}}
	_, err := uc.Execute(context.Background(), request.ConvertSalesQuotationDTO{Code: 1})
	if err == nil || !strings.Contains(err.Error(), "inativo") {
		t.Fatalf("converteu orçamento de cliente inativo: %v", err)
	}
	if pedidos.statusNaCriacao != "" {
		t.Fatal("criou o pedido antes de recusar")
	}
	uc.Quotes.Customers = clienteSituacaoFake{ativo: true}
	if _, err := uc.Execute(context.Background(), request.ConvertSalesQuotationDTO{Code: 1}); err != nil {
		t.Fatalf("cliente ativo devia converter: %v", err)
	}
}
