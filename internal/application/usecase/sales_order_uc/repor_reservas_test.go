package sales_order_uc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

type estoqueComSaldo struct {
	reservasRefeitas
	disponivel decimal.Decimal
}

func (e *estoqueComSaldo) GetBalance(context.Context, int64, string, int64) (*stockentity.StockBalance, error) {
	return &stockentity.StockBalance{AvailableQty: e.disponivel}, nil
}

func reposicaoDe(status entity.SalesOrderStatus, bloqueado bool, linhas []*entity.SalesOrderItem, e *estoqueComSaldo) *ReposicaoDeReservas {
	repo := &fakeSORepo{order: &entity.SalesOrder{Code: 7, Status: status, IsBlocked: bloqueado}, items: linhas}
	return &ReposicaoDeReservas{Transacao: func(ctx context.Context, fn func(ConfirmacaoRepos) error) error {
		return fn(ConfirmacaoRepos{Pedidos: repo, Estoque: e})
	}}
}

func linhaAberta(code, produto int64, pedida, atendida string) *entity.SalesOrderItem {
	dep := int64(2)
	st := entity.SalesOrderItemStatusOpen
	if dec(atendida).IsPositive() {
		st = entity.SalesOrderItemStatusPartial
	}
	return &entity.SalesOrderItem{Code: code, SalesOrderCode: 7, ItemCode: produto, WarehouseCode: &dep, IsActive: true,
		Status: st, RequestedQty: dec(pedida), AttendedQty: dec(atendida)}
}

// A NF-e cancelada devolve o saldo à linha: a reserva completa o que falta,
// contando a que ainda está ativa, e não passa do disponível.
func TestNotaCanceladaRepoeAReservaDoPedido(t *testing.T) {
	ativa := reservaDe(1, 7, 10002, ref(501))
	ativa.Quantity = dec("30")
	e := &estoqueComSaldo{disponivel: dec("1000")}
	e.ativas = []*stockentity.StockReservation{ativa}
	linhas := []*entity.SalesOrderItem{linhaAberta(501, 10002, "100", "0"), linhaAberta(502, 20000, "40", "10")}
	if err := reposicaoDe(entity.SalesOrderStatusOrder, false, linhas, e).ReporReservasDoPedido(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if len(e.criadas) != 2 || !e.criadas[0].Quantity.Equal(dec("70")) || *e.criadas[0].ReferenceItemCode != 501 ||
		!e.criadas[1].Quantity.Equal(dec("30")) || *e.criadas[1].ReferenceItemCode != 502 || e.criadas[1].ReferenceCode != 7 {
		t.Fatalf("criadas = %+v", e.criadas)
	}

	e = &estoqueComSaldo{disponivel: dec("25")}
	if err := reposicaoDe(entity.SalesOrderStatusOrder, false, linhas[:1], e).ReporReservasDoPedido(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if len(e.criadas) != 1 || !e.criadas[0].Quantity.Equal(dec("25")) {
		t.Fatalf("limitada ao disponível: %+v", e.criadas)
	}
}

// A reserva da onda guarda o produto: ela conta para as linhas desse produto,
// uma vez só.
func TestReposicaoContaAReservaComCodigoDoProduto(t *testing.T) {
	onda := reservaDe(1, 7, 10002, ref(10002))
	onda.Quantity = dec("50")
	e := &estoqueComSaldo{disponivel: dec("1000")}
	e.ativas = []*stockentity.StockReservation{onda}
	linhas := []*entity.SalesOrderItem{linhaAberta(501, 10002, "40", "0"), linhaAberta(502, 10002, "40", "0")}
	if err := reposicaoDe(entity.SalesOrderStatusOrder, false, linhas, e).ReporReservasDoPedido(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if len(e.criadas) != 1 || *e.criadas[0].ReferenceItemCode != 502 || !e.criadas[0].Quantity.Equal(dec("30")) {
		t.Fatalf("80 pedidos, 50 já reservados: esperado 30 na linha 502, criadas = %+v", e.criadas)
	}
}

func TestReposicaoIgnoraPedidoForaDaCarteira(t *testing.T) {
	casos := []struct {
		status    entity.SalesOrderStatus
		bloqueado bool
	}{
		{entity.SalesOrderStatusCancelled, false},
		{entity.SalesOrderStatusDraft, false},
		{entity.SalesOrderStatusInvoiced, false},
		{entity.SalesOrderStatusOrder, true},
	}
	for _, c := range casos {
		e := &estoqueComSaldo{disponivel: dec("1000")}
		linhas := []*entity.SalesOrderItem{linhaAberta(501, 10002, "100", "0")}
		if err := reposicaoDe(c.status, c.bloqueado, linhas, e).ReporReservasDoPedido(context.Background(), 7); err != nil {
			t.Fatal(err)
		}
		if len(e.criadas) != 0 {
			t.Fatalf("%s bloqueado=%v reservou %+v", c.status, c.bloqueado, e.criadas)
		}
	}
}

// Linha que aumentou: só ela recebe o acréscimo da reserva.
func TestReposicaoDeUmaLinhaSo(t *testing.T) {
	e := &estoqueComSaldo{disponivel: dec("1000")}
	linhas := []*entity.SalesOrderItem{linhaAberta(501, 10002, "100", "0"), linhaAberta(502, 20000, "40", "0")}
	repos := ConfirmacaoRepos{Pedidos: &fakeSORepo{order: &entity.SalesOrder{Code: 7, Status: entity.SalesOrderStatusOrder}, items: linhas}, Estoque: e}
	if err := reporReservas(context.Background(), repos, 7, 502); err != nil {
		t.Fatal(err)
	}
	if len(e.criadas) != 1 || *e.criadas[0].ReferenceItemCode != 502 || !e.criadas[0].Quantity.Equal(dec("40")) {
		t.Fatalf("criadas = %+v", e.criadas)
	}
}

func TestCampoOpcionalAusenteMantemOValor(t *testing.T) {
	atual, vazio, novo := "LOTE-1", "", "LOTE-2"
	if v := manterSeAusente(nil, &atual); v == nil || *v != "LOTE-1" {
		t.Fatalf("ausente: %v", v)
	}
	if v := manterSeAusente(&vazio, &atual); v != nil {
		t.Fatalf("vazio limpa: %v", *v)
	}
	if v := manterSeAusente(&novo, &atual); v == nil || *v != "LOTE-2" {
		t.Fatalf("novo: %v", v)
	}
}
