package sales_quotation_uc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	quoteentity "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	quoterepo "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository"
)

type conversaoRepo struct {
	quoterepo.SalesQuotationRepository
	q     *quoteentity.SalesQuotation
	itens []*quoteentity.SalesQuotationItem
}

func (r conversaoRepo) GetByCode(context.Context, int64) (*quoteentity.SalesQuotation, error) {
	return r.q, nil
}
func (r conversaoRepo) ListItems(context.Context, int64) ([]*quoteentity.SalesQuotationItem, error) {
	return r.itens, nil
}

type pedidosConvertidos struct {
	orderrepo.SalesOrderRepository
	itens []*orderentity.SalesOrderItem
}

func (p *pedidosConvertidos) NextOrderNumber(context.Context, int64) (int64, error) { return 1, nil }
func (p *pedidosConvertidos) Create(_ context.Context, o *orderentity.SalesOrder) (*orderentity.SalesOrder, error) {
	o.Code = 77
	return o, nil
}
func (p *pedidosConvertidos) CreateItem(_ context.Context, it *orderentity.SalesOrderItem) (*orderentity.SalesOrderItem, error) {
	p.itens = append(p.itens, it)
	return it, nil
}
func (p *pedidosConvertidos) Update(_ context.Context, o *orderentity.SalesOrder) (*orderentity.SalesOrder, error) {
	return o, nil
}

type uowDireta struct{ pedidos *pedidosConvertidos }

func (u uowDireta) Execute(_ context.Context, _ int64, fn func(orderrepo.SalesOrderRepository) (*orderentity.SalesOrder, error)) (*orderentity.SalesOrder, error) {
	return fn(u.pedidos)
}

type sugestorConversao struct{}

func (sugestorConversao) SugerirTributos(context.Context, ports.ConsultaTributosVenda) (*ports.TributosSugeridos, error) {
	return &ports.TributosSugeridos{IPIPct: decimal.NewFromInt(9), ICMSPct: decimal.RequireFromString("19.5"),
		PISPct: decimal.RequireFromString("1.65"), COFINSPct: decimal.RequireFromString("7.6")}, nil
}

// Na conversão, IPI e ST seguem como negociados no orçamento; ICMS, PIS e
// COFINS (que o orçamento não guarda) vêm da regra, como a nota vai calcular.
func TestConversaoPreencheICMSPISCOFINSDaRegra(t *testing.T) {
	cliente := int64(5)
	q := &quoteentity.SalesQuotation{Code: 1, CustomerCode: &cliente, Status: quoteentity.SalesQuotationStatusDraft}
	itens := []*quoteentity.SalesQuotationItem{{Code: 10, ItemCode: 99, RequestedQty: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100),
		IPIPct: decimal.NewFromInt(5), STPct: decimal.NewFromInt(2), IsActive: true, Status: quoteentity.SalesQuotationItemStatusOpen}}
	pedidos := &pedidosConvertidos{}
	uc := &ConvertUseCase{Quotes: &UseCase{Repo: conversaoRepo{q: q, itens: itens}, Auth: quotationRepresentativeAuth{}, Tributos: sugestorConversao{}},
		UOW: uowDireta{pedidos}}
	if _, err := uc.Execute(context.Background(), request.ConvertSalesQuotationDTO{Code: 1}); err != nil {
		t.Fatal(err)
	}
	if len(pedidos.itens) != 1 {
		t.Fatalf("linhas convertidas: %d", len(pedidos.itens))
	}
	l := pedidos.itens[0]
	if l.IPIPct != 5 || l.STPct != 2 || l.ICMSPct != 19.5 || l.PISPct != 1.65 || l.COFINSPct != 7.6 {
		t.Fatalf("linha do pedido: IPI %v ST %v ICMS %v PIS %v COFINS %v", l.IPIPct, l.STPct, l.ICMSPct, l.PISPct, l.COFINSPct)
	}
}
