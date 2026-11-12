package sales_order_uc

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

// ReposicaoDeReservas devolve ao pedido a reserva que a NF-e consumiu quando a
// nota é cancelada. O cancelamento devolve a mercadoria ao estoque e o saldo à
// linha do pedido, mas a reserva consumida não voltava: o pedido seguia na
// carteira sem estoque prometido, e a mercadoria devolvida ficava livre para
// outro pedido. A reserva confirmada (OrderStockReserver.Reserve) não serve
// aqui: ela não mexe em pedido que ainda tenha alguma reserva ativa.
type ReposicaoDeReservas struct {
	Transacao EmTransacao
}

// ReporReservasDoPedido completa, linha a linha, a reserva do pedido até o
// saldo em aberto, limitada ao disponível. Só pedido confirmado e liberado:
// rascunho e orçamento ainda não reservam, e o bloqueado reserva no desbloqueio.
func (r *ReposicaoDeReservas) ReporReservasDoPedido(ctx context.Context, code int64) error {
	if r == nil || r.Transacao == nil {
		return nil
	}
	return r.Transacao(ctx, func(repos ConfirmacaoRepos) error {
		return reporReservas(ctx, repos, code, 0)
	})
}

// soLinha > 0 limita a reposição a essa linha.
func reporReservas(ctx context.Context, repos ConfirmacaoRepos, code, soLinha int64) error {
	if repos.Estoque == nil {
		return nil
	}
	pedido, err := repos.Pedidos.GetByCode(ctx, code)
	if err != nil {
		return fmt.Errorf("lendo o pedido %d: %w", code, err)
	}
	if pedido == nil || pedido.Status != entity.SalesOrderStatusOrder || pedido.IsBlocked {
		return nil
	}
	linhas, err := repos.Pedidos.ListItems(ctx, code)
	if err != nil {
		return fmt.Errorf("lendo os itens do pedido %d: %w", code, err)
	}
	ativas, err := repos.Estoque.ListActiveReservations(ctx)
	if err != nil {
		return fmt.Errorf("lendo as reservas do pedido %d: %w", code, err)
	}
	reservado := reservadoPorLinha(code, linhas, ativas)
	for _, it := range linhas {
		if (soLinha > 0 && it.Code != soLinha) || !it.IsActive || it.WarehouseCode == nil ||
			(it.Status != entity.SalesOrderItemStatusOpen && it.Status != entity.SalesOrderItemStatusPartial) {
			continue
		}
		aberto := it.RequestedQty.Sub(it.AttendedQty).Sub(it.CancelledQty)
		falta := aberto.Sub(reservado.daLinha(it))
		if !falta.IsPositive() {
			continue
		}
		if err := reservarDisponivel(ctx, repos.Estoque, pedido, it, falta); err != nil {
			return err
		}
	}
	return nil
}

// reservasDoPedidoPorLinha soma as reservas ativas do pedido por linha. A
// reserva que aponta o código da linha é dela; a antiga e a da onda (com o
// código do produto ou sem referência) vão para um saldo do produto, gasto
// pelas linhas desse produto na ordem do pedido — assim nada é reservado duas
// vezes.
type reservasDoPedidoPorLinha struct {
	linha   map[int64]decimal.Decimal
	produto map[int64]decimal.Decimal
}

func reservadoPorLinha(code int64, linhas []*entity.SalesOrderItem, ativas []*stockentity.StockReservation) *reservasDoPedidoPorLinha {
	out := &reservasDoPedidoPorLinha{linha: map[int64]decimal.Decimal{}, produto: map[int64]decimal.Decimal{}}
	produtoDaLinha := map[int64]int64{}
	for _, it := range linhas {
		produtoDaLinha[it.Code] = it.ItemCode
	}
	for _, r := range ativas {
		if r.ReferenceType != stockentity.ReferenceTypeSalesOrder || r.ReferenceCode != code {
			continue
		}
		if r.ReferenceItemCode != nil {
			if produto, ok := produtoDaLinha[*r.ReferenceItemCode]; ok && produto == r.ItemCode {
				out.linha[*r.ReferenceItemCode] = out.linha[*r.ReferenceItemCode].Add(r.Quantity)
				continue
			}
		}
		out.produto[r.ItemCode] = out.produto[r.ItemCode].Add(r.Quantity)
	}
	return out
}

// daLinha devolve o reservado da linha, usando o saldo do produto que ainda
// não foi atribuído a outra linha.
func (s *reservasDoPedidoPorLinha) daLinha(it *entity.SalesOrderItem) decimal.Decimal {
	total := s.linha[it.Code]
	aberto := it.RequestedQty.Sub(it.AttendedQty).Sub(it.CancelledQty)
	if falta := aberto.Sub(total); falta.IsPositive() {
		usa := decimal.Min(falta, s.produto[it.ItemCode])
		s.produto[it.ItemCode] = s.produto[it.ItemCode].Sub(usa)
		total = total.Add(usa)
	}
	return total
}
