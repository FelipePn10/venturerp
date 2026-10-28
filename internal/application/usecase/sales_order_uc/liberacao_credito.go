package sales_order_uc

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/aprovacao"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
)

// Aprovacoes é o motor de aprovação visto pelo pedido de venda.
type Aprovacoes interface {
	Solicitar(ctx context.Context, t aprovacao.Tipo, documento int64, valor decimal.Decimal, descricao string) (*aprovacao.Solicitacao, error)
	PendenteDoDocumento(ctx context.Context, t aprovacao.Tipo, documento int64) (*aprovacao.Solicitacao, error)
}

// ConclusaoDoDesbloqueio desbloqueia o pedido e, se ele já está confirmado,
// termina a confirmação que o bloqueio interrompeu: demanda do MRP e reserva de
// estoque. Antes o desbloqueio só limpava o sinal — o pedido liberado nunca
// entrava no MRP nem reservava estoque. As duas etapas são idempotentes.
type ConclusaoDoDesbloqueio struct {
	Transacao  EmTransacao
	ComDemanda bool
	Reserver   *OrderStockReserver
}

func (c *ConclusaoDoDesbloqueio) Concluir(ctx context.Context, code int64) error {
	if c.Transacao == nil {
		return fmt.Errorf("desbloqueio do pedido sem transação configurada")
	}
	return c.Transacao(ctx, func(r ConfirmacaoRepos) error {
		order, err := r.Pedidos.GetByCode(ctx, code)
		if err != nil {
			return err
		}
		if order.IsBlocked {
			if err := r.Pedidos.Unblock(ctx, code); err != nil {
				return err
			}
		}
		if order.Status != entity.SalesOrderStatusOrder {
			return nil
		}
		if c.ComDemanda && r.Demandas != nil {
			if err := generateDemands(ctx, code, r.Pedidos, r.Demandas); err != nil {
				return err
			}
		}
		if c.Reserver != nil && r.Estoque != nil {
			if err := c.Reserver.Reserve(ctx, code, r.Pedidos, r.Estoque); err != nil {
				return err
			}
		}
		return nil
	})
}

// bloqueiosPorAprovacao: os tipos de aprovação que bloqueiam o pedido de
// venda. O pedido tem um único sinal de bloqueio, então ele só é liberado
// quando nenhum deles está pendente.
var bloqueiosPorAprovacao = []aprovacao.Tipo{aprovacao.LiberacaoCredito, aprovacao.PedidoPoliticaComercial}

// outraPendente devolve a aprovação pendente do pedido que não seja do tipo
// `exceto` (vazio = qualquer uma).
func outraPendente(ctx context.Context, a Aprovacoes, code int64, exceto aprovacao.Tipo) (*aprovacao.Solicitacao, error) {
	for _, t := range bloqueiosPorAprovacao {
		if t == exceto {
			continue
		}
		p, err := a.PendenteDoDocumento(ctx, t, code)
		if err != nil || p != nil {
			return p, err
		}
	}
	return nil, nil
}

// LiberacaoDeCredito é o efetivador das aprovações que bloqueiam o pedido
// (LIBERACAO_CREDITO e PEDIDO_POLITICA_COMERCIAL).
type LiberacaoDeCredito struct {
	Conclusao *ConclusaoDoDesbloqueio
	// Tipo e Aprovacoes: com OUTRA aprovação ainda pendente no mesmo pedido, a
	// aprovação deste tipo conclui mas o pedido segue bloqueado — quem libera é
	// a última. Sem eles, desbloqueia direto, como antes.
	Tipo       aprovacao.Tipo
	Aprovacoes Aprovacoes
}

func (l *LiberacaoDeCredito) Efetivar(ctx context.Context, code, _ int64) error {
	if l.Aprovacoes != nil {
		p, err := outraPendente(ctx, l.Aprovacoes, code, l.Tipo)
		if err != nil {
			return err
		}
		if p != nil {
			return nil
		}
	}
	return l.Conclusao.Concluir(ctx, code)
}

// Reprovar mantém o pedido bloqueado; o motivo fica no histórico da aprovação.
func (l *LiberacaoDeCredito) Reprovar(context.Context, int64, int64, string) error { return nil }

// solicitarLiberacao abre a aprovação de crédito do pedido recém-bloqueado.
func solicitarLiberacao(ctx context.Context, a Aprovacoes, pedidos interface {
	GetByCode(context.Context, int64) (*entity.SalesOrder, error)
}, code int64) error {
	order, err := pedidos.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	motivo := "pedido bloqueado pelo limite de crédito"
	if order.BlockReason != nil && *order.BlockReason != "" {
		motivo = *order.BlockReason
	}
	if _, err := a.Solicitar(ctx, aprovacao.LiberacaoCredito, code, orderExposureValue(order), motivo); err != nil {
		return errorsuc.NewValidationError(fmt.Sprintf("o pedido %d foi bloqueado pelo crédito, mas a aprovação não pôde ser aberta (%s): tente confirmar de novo", code, err.Error()))
	}
	return nil
}
