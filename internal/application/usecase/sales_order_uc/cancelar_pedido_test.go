package sales_order_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type pedidoCancelavel struct {
	*fakeSORepo
	cancelado, linhasFechadas bool
}

func (p *pedidoCancelavel) Cancel(context.Context, int64, string, *string) error {
	p.cancelado = true
	return nil
}
func (p *pedidoCancelavel) CancelarLinhasEmAberto(context.Context, int64) error {
	p.linhasFechadas = true
	return nil
}

type reservasEmMemoria struct {
	stockrepo.StockRepository
	ativas     []*stockentity.StockReservation
	canceladas []int64
}

func (r *reservasEmMemoria) ListActiveReservations(context.Context) ([]*stockentity.StockReservation, error) {
	return r.ativas, nil
}
func (r *reservasEmMemoria) CancelReservation(_ context.Context, id int64) error {
	r.canceladas = append(r.canceladas, id)
	return nil
}

func cancelamentoDe(status entity.SalesOrderStatus, estoque *reservasEmMemoria) (*pedidoCancelavel, *CancelSalesOrderUseCase) {
	repo := &pedidoCancelavel{fakeSORepo: &fakeSORepo{order: &entity.SalesOrder{Code: 7, Status: status}}}
	uc := &CancelSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true},
		Transacao: func(ctx context.Context, fn func(ConfirmacaoRepos) error) error {
			return fn(ConfirmacaoRepos{Pedidos: repo, Estoque: estoque})
		}}
	return repo, uc
}

// Cancelar o pedido fecha o saldo das linhas e solta só as reservas dele.
func TestCancelarPedidoFechaLinhasESoltaAsReservas(t *testing.T) {
	estoque := &reservasEmMemoria{ativas: []*stockentity.StockReservation{
		{ID: 1, ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: 7},
		{ID: 2, ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: 8}, // outro pedido
		{ID: 3, ReferenceType: "SHIPMENT", ReferenceCode: 7},                          // romaneio de mesmo número
		{ID: 4, ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: 7},
	}}
	repo, uc := cancelamentoDe(entity.SalesOrderStatusOrder, estoque)
	if err := uc.Execute(context.Background(), request.CancelSalesOrderDTO{Code: 7, Reason: "cliente desistiu"}); err != nil {
		t.Fatal(err)
	}
	if !repo.cancelado || !repo.linhasFechadas {
		t.Fatalf("cancelado=%v linhas fechadas=%v", repo.cancelado, repo.linhasFechadas)
	}
	if len(estoque.canceladas) != 2 || estoque.canceladas[0] != 1 || estoque.canceladas[1] != 4 {
		t.Fatalf("reservas liberadas = %v, esperado [1 4]", estoque.canceladas)
	}
}

func TestNaoCancelaPedidoFaturadoOuJaCancelado(t *testing.T) {
	for status, trecho := range map[entity.SalesOrderStatus]string{
		entity.SalesOrderStatusInvoiced:  "cancele a NF-e",
		entity.SalesOrderStatusCancelled: "já está cancelado",
	} {
		repo, uc := cancelamentoDe(status, &reservasEmMemoria{})
		err := uc.Execute(context.Background(), request.CancelSalesOrderDTO{Code: 7, Reason: "teste"})
		if err == nil || !strings.Contains(err.Error(), trecho) {
			t.Fatalf("%s: erro = %v, esperado conter %q", status, err, trecho)
		}
		if repo.cancelado {
			t.Fatalf("%s: o pedido foi cancelado", status)
		}
	}
}
