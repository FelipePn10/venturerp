package sales_order_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

type linhaCancelavel struct {
	*fakeSORepo
	linha      *entity.SalesOrderItem
	cancelada  bool
	recalculou bool
}

func (l *linhaCancelavel) GetItem(context.Context, int64) (*entity.SalesOrderItem, error) {
	return l.linha, nil
}
func (l *linhaCancelavel) CancelItem(context.Context, int64) error {
	l.cancelada = true
	return nil
}
func (l *linhaCancelavel) RecalculateTotals(context.Context, int64) error {
	l.recalculou = true
	return nil
}

func linhaDe(pedido entity.SalesOrderStatus, linha entity.SalesOrderItemStatus, estoque *reservasEmMemoria) (*linhaCancelavel, *CancelSalesOrderItemUseCase) {
	repo := &linhaCancelavel{
		fakeSORepo: &fakeSORepo{order: &entity.SalesOrder{Code: 7, Status: pedido}},
		linha:      &entity.SalesOrderItem{Code: 501, SalesOrderCode: 7, ItemCode: 10002, Status: linha},
	}
	uc := &CancelSalesOrderItemUseCase{Repo: repo, Auth: fakeAuth{can: true},
		Transacao: func(ctx context.Context, fn func(ConfirmacaoRepos) error) error {
			return fn(ConfirmacaoRepos{Pedidos: repo, Estoque: estoque})
		}}
	return repo, uc
}

func ref(v int64) *int64 { return &v }

func reservaDe(id, pedido, produto int64, refLinha *int64) *stockentity.StockReservation {
	return &stockentity.StockReservation{ID: id, ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: pedido, ItemCode: produto, ReferenceItemCode: refLinha}
}

// Cancelar a linha solta só a reserva dela; a das outras linhas fica.
func TestCancelarLinhaSoltaSoAReservaDaLinha(t *testing.T) {
	estoque := &reservasEmMemoria{ativas: []*stockentity.StockReservation{
		reservaDe(1, 7, 10002, ref(501)),
		reservaDe(2, 7, 20000, ref(502)), // outra linha
		reservaDe(3, 8, 10002, ref(501)), // outro pedido
		reservaDe(4, 7, 30000, ref(501)), // número da linha, produto de outra (a onda grava o produto)
	}}
	repo, uc := linhaDe(entity.SalesOrderStatusOrder, entity.SalesOrderItemStatusPartial, estoque)
	if err := uc.Execute(context.Background(), 501); err != nil {
		t.Fatal(err)
	}
	if !repo.cancelada || !repo.recalculou {
		t.Fatalf("cancelada=%v recalculou=%v", repo.cancelada, repo.recalculou)
	}
	if len(estoque.canceladas) != 1 || estoque.canceladas[0] != 1 {
		t.Fatalf("reservas liberadas = %v, esperado [1]", estoque.canceladas)
	}
}

// Reserva antiga (com o código do produto no lugar da linha) só é da linha
// quando o produto aparece numa linha só do pedido.
func TestReservaAntigaComCodigoDoProduto(t *testing.T) {
	antigas := func() *reservasEmMemoria {
		return &reservasEmMemoria{ativas: []*stockentity.StockReservation{reservaDe(9, 7, 10002, ref(10002)), reservaDe(10, 7, 10002, nil)}}
	}
	estoque := antigas()
	_, uc := linhaDe(entity.SalesOrderStatusOrder, entity.SalesOrderItemStatusOpen, estoque)
	if err := uc.Execute(context.Background(), 501); err != nil {
		t.Fatal(err)
	}
	if len(estoque.canceladas) != 2 {
		t.Fatalf("produto numa linha só: liberadas %v, esperado [9 10]", estoque.canceladas)
	}
	estoque = antigas()
	repo, uc := linhaDe(entity.SalesOrderStatusOrder, entity.SalesOrderItemStatusOpen, estoque)
	repo.items = []*entity.SalesOrderItem{{Code: 502, ItemCode: 10002, IsActive: true}}
	if err := uc.Execute(context.Background(), 501); err != nil {
		t.Fatal(err)
	}
	if len(estoque.canceladas) != 0 {
		t.Fatalf("produto em duas linhas: não dá para saber de qual é a reserva, liberou %v", estoque.canceladas)
	}
}

func TestNaoCancelaLinhaSemSaldoOuDePedidoEncerrado(t *testing.T) {
	casos := []struct {
		pedido entity.SalesOrderStatus
		linha  entity.SalesOrderItemStatus
		trecho string
	}{
		{entity.SalesOrderStatusOrder, entity.SalesOrderItemStatusCancelled, "já está cancelada"},
		{entity.SalesOrderStatusOrder, entity.SalesOrderItemStatusDelivered, "atendida por inteiro"},
		{entity.SalesOrderStatusInvoiced, entity.SalesOrderItemStatusOpen, "cancele a NF-e"},
		{entity.SalesOrderStatusCancelled, entity.SalesOrderItemStatusOpen, "está cancelado"},
	}
	for _, c := range casos {
		repo, uc := linhaDe(c.pedido, c.linha, &reservasEmMemoria{})
		err := uc.Execute(context.Background(), 501)
		if err == nil || !strings.Contains(err.Error(), c.trecho) {
			t.Fatalf("%s/%s: erro = %v, esperado conter %q", c.pedido, c.linha, err, c.trecho)
		}
		if repo.cancelada {
			t.Fatalf("%s/%s: a linha foi cancelada", c.pedido, c.linha)
		}
	}
}
