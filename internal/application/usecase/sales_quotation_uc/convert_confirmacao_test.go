package sales_quotation_uc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	quoteentity "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
)

type confirmadorFake struct {
	chamadas []request.ChangeStatusDTO
	erro     error
}

func (c *confirmadorFake) Execute(_ context.Context, dto request.ChangeStatusDTO) error {
	c.chamadas = append(c.chamadas, dto)
	return c.erro
}

type pedidosComStatus struct {
	pedidosConvertidos
	statusNaCriacao orderentity.SalesOrderStatus
}

func (p *pedidosComStatus) Create(ctx context.Context, o *orderentity.SalesOrder) (*orderentity.SalesOrder, error) {
	p.statusNaCriacao = o.Status
	return p.pedidosConvertidos.Create(ctx, o)
}

// Converter direto para pedido passa pela confirmação (crédito, pedido de
// compra, linhas, reserva): nasce rascunho e é confirmado depois.
func TestConversaoPassaPelaConfirmacao(t *testing.T) {
	cliente := int64(5)
	q := &quoteentity.SalesQuotation{Code: 1, CustomerCode: &cliente, Status: quoteentity.SalesQuotationStatusDraft}
	itens := []*quoteentity.SalesQuotationItem{{Code: 10, ItemCode: 99, RequestedQty: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100),
		IsActive: true, Status: quoteentity.SalesQuotationItemStatusOpen}}

	pedidos := &pedidosComStatus{}
	conf := &confirmadorFake{}
	uc := &ConvertUseCase{Quotes: &UseCase{Repo: conversaoRepo{q: q, itens: itens}, Auth: quotationRepresentativeAuth{}, Tributos: sugestorConversao{}},
		UOW: uowPedidos{pedidos}, Confirmar: conf}
	r, err := uc.Execute(context.Background(), request.ConvertSalesQuotationDTO{Code: 1})
	if err != nil {
		t.Fatal(err)
	}
	if pedidos.statusNaCriacao != orderentity.SalesOrderStatusDraft {
		t.Fatalf("o pedido devia nascer rascunho, nasceu %q", pedidos.statusNaCriacao)
	}
	if len(conf.chamadas) != 1 || conf.chamadas[0].Code != 77 || conf.chamadas[0].Status != "P" || r.Status != "P" {
		t.Fatalf("confirmação: %+v resposta %+v", conf.chamadas, r)
	}

	// Confirmação recusada: o erro diz o número do pedido e o motivo.
	conf2 := &confirmadorFake{erro: errors.New("o cliente 5 exige o número do pedido de compra dele")}
	uc.Confirmar = conf2
	_, err = uc.Execute(context.Background(), request.ConvertSalesQuotationDTO{Code: 1})
	if err == nil || !strings.Contains(err.Error(), "pedido 77") || !strings.Contains(err.Error(), "pedido de compra") {
		t.Fatalf("erro da confirmação: %v", err)
	}

	// Status explícito diferente de pedido (ex.: orçamento firme) não confirma.
	conf3 := &confirmadorFake{}
	uc.Confirmar = conf3
	if _, err := uc.Execute(context.Background(), request.ConvertSalesQuotationDTO{Code: 1, Status: "OF"}); err != nil {
		t.Fatal(err)
	}
	if len(conf3.chamadas) != 0 {
		t.Fatal("status OF não passa pela confirmação")
	}
}

type uowPedidos struct{ pedidos *pedidosComStatus }

func (u uowPedidos) Execute(_ context.Context, _ int64, fn func(orderrepo.SalesOrderRepository) (*orderentity.SalesOrder, error)) (*orderentity.SalesOrder, error) {
	return fn(u.pedidos)
}
