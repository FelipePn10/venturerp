package sales_order_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

var dec = decimal.RequireFromString

// O corpo da alteração não muda o atendido (vem das notas) nem desfaz
// cancelamento, e a quantidade não fica abaixo do atendido + cancelado.
func TestAlteracaoDaLinhaNaoMexeNoAtendido(t *testing.T) {
	atual := &entity.SalesOrderItem{RequestedQty: dec("100"), AttendedQty: dec("30"), CancelledQty: dec("10")}
	dto := request.UpdateSalesOrderItemDTO{RequestedQty: dec("80"), AttendedQty: dec("80"), CancelledQty: dec("0")}
	if err := quantidadesDaAlteracao(atual, &dto); err != nil {
		t.Fatal(err)
	}
	if !dto.AttendedQty.Equal(dec("30")) || !dto.CancelledQty.Equal(dec("10")) {
		t.Fatalf("atendido=%s cancelado=%s, esperado 30 e 10", dto.AttendedQty, dto.CancelledQty)
	}
	dto = request.UpdateSalesOrderItemDTO{RequestedQty: dec("35")}
	if err := quantidadesDaAlteracao(atual, &dto); err == nil || !strings.Contains(err.Error(), "abaixo do já atendido") {
		t.Fatalf("35 < 30 + 10: erro = %v", err)
	}
	dto = request.UpdateSalesOrderItemDTO{RequestedQty: dec("0")}
	if err := quantidadesDaAlteracao(atual, &dto); err == nil {
		t.Fatal("aceitou quantidade zero")
	}
}

type reservasRefeitas struct {
	reservasEmMemoria
	criadas []*stockentity.StockReservation
}

func (r *reservasRefeitas) CreateReservation(_ context.Context, res *stockentity.StockReservation) (*stockentity.StockReservation, error) {
	r.criadas = append(r.criadas, res)
	return res, nil
}

// A linha que diminuiu solta a reserva antiga e fica reservada só pelo saldo.
func TestLinhaQueDiminuiAparaAReserva(t *testing.T) {
	linha := &entity.SalesOrderItem{Code: 501, SalesOrderCode: 7, ItemCode: 10002}
	nova := func() *reservasRefeitas {
		r := reservaDe(1, 7, 10002, ref(501))
		r.Quantity, r.WarehouseID = dec("100"), 2
		return &reservasRefeitas{reservasEmMemoria: reservasEmMemoria{ativas: []*stockentity.StockReservation{r}}}
	}
	repos := func(e *reservasRefeitas) ConfirmacaoRepos {
		return ConfirmacaoRepos{Pedidos: &fakeSORepo{}, Estoque: e}
	}

	estoque := nova()
	if err := aparaReservaDaLinha(context.Background(), repos(estoque), linha, dec("60")); err != nil {
		t.Fatal(err)
	}
	if len(estoque.canceladas) != 1 || len(estoque.criadas) != 1 || !estoque.criadas[0].Quantity.Equal(dec("60")) ||
		estoque.criadas[0].WarehouseID != 2 || *estoque.criadas[0].ReferenceItemCode != 501 {
		t.Fatalf("canceladas=%v criadas=%+v", estoque.canceladas, estoque.criadas)
	}
	estoque = nova()
	if err := aparaReservaDaLinha(context.Background(), repos(estoque), linha, dec("120")); err != nil {
		t.Fatal(err)
	}
	if len(estoque.canceladas) != 0 || len(estoque.criadas) != 0 {
		t.Fatalf("linha que aumentou mexeu na reserva: %v %v", estoque.canceladas, estoque.criadas)
	}
	estoque = nova()
	if err := aparaReservaDaLinha(context.Background(), repos(estoque), linha, dec("0")); err != nil {
		t.Fatal(err)
	}
	if len(estoque.canceladas) != 1 || len(estoque.criadas) != 0 {
		t.Fatalf("sem saldo em aberto: canceladas=%v criadas=%v", estoque.canceladas, estoque.criadas)
	}
}
