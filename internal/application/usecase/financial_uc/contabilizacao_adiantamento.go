package financial_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// paramsDoAdiantamento devolve os parâmetros quando a contabilização do lado
// do adiantamento está ligada: pago a fornecedor segue "contabilizar
// pagamentos"; recebido de cliente, "contabilizar recebimentos".
func paramsDoAdiantamento(ctx context.Context, c Contabil, tipo entity.AdiantamentoTipo) (*fiscalrepo.AccountingPostingParams, error) {
	if c == nil {
		return nil, nil
	}
	p, err := c.AccountingParams(ctx)
	if err != nil || p == nil {
		return nil, err
	}
	if tipo == entity.AdiantamentoTipoPagar && !p.ContabilizarPagamentos {
		return nil, nil
	}
	if tipo == entity.AdiantamentoTipoReceber && !p.ContabilizarRecebimentos {
		return nil, nil
	}
	return p, nil
}

func contaDoAdiantamento(p *fiscalrepo.AccountingPostingParams, tipo entity.AdiantamentoTipo) *int64 {
	if tipo == entity.AdiantamentoTipoPagar {
		return p.AdiantamentoFornecedoresAccountID
	}
	return p.AdiantamentoClientesAccountID
}

func historicoAdiantamento(a *entity.Adiantamento) string {
	h := "Adiantamento recebido de cliente"
	if a.Tipo == entity.AdiantamentoTipoPagar {
		h = "Adiantamento pago a fornecedor"
	}
	if a.NumeroDocumento != nil && *a.NumeroDocumento != "" {
		h += " — documento " + *a.NumeroDocumento
	}
	return h
}

// loteAdiantamento monta a entrada do adiantamento; nil quando desligada.
// O SourceID (id do adiantamento) é preenchido pelo repositório após o INSERT.
func loteAdiantamento(ctx context.Context, c Contabil, a *entity.Adiantamento) (*contabilizacao.Lote, error) {
	p, err := paramsDoAdiantamento(ctx, c, a.Tipo)
	if err != nil || p == nil {
		return nil, err
	}
	banco, err := baixaContabil{c}.banco(ctx, p, a.ContaBancariaID)
	if err != nil {
		return nil, err
	}
	ls, faltando := contabilizacao.Adiantamento(a.Tipo == entity.AdiantamentoTipoPagar, banco, contaDoAdiantamento(p, a.Tipo), a.ValorOriginal, historicoAdiantamento(a))
	if len(faltando) > 0 {
		return nil, faltaConta("dos adiantamentos", faltando)
	}
	if err := contasValidas(ctx, c, p.PlanID, ls); err != nil {
		return nil, err
	}
	return &contabilizacao.Lote{PlanID: p.PlanID, Data: a.DataAdiantamento, Prefixo: "ADT", SourceType: contabilizacao.OrigemAdiantamento, Lancamentos: ls}, nil
}

// loteAplicacao transfere o valor aplicado da conta do adiantamento para a
// contrapartida do título (a mesma que a baixa usaria). Só contabiliza se a
// entrada do adiantamento foi contabilizada: sem ela a conta do adiantamento
// não tem saldo e ficaria negativa. O SourceID (id da aplicação) é
// preenchido pelo repositório.
func loteAplicacao(ctx context.Context, c Contabil, a *entity.Adiantamento, cp *entity.ContaPagar, cr *entity.ContaReceber, valor decimal.Decimal, data time.Time) (*contabilizacao.Lote, error) {
	p, err := paramsDoAdiantamento(ctx, c, a.Tipo)
	if err != nil || p == nil {
		return nil, err
	}
	entrou, err := c.Contabilizado(ctx, contabilizacao.OrigemAdiantamento, a.ID)
	if err != nil || !entrou {
		return nil, err
	}
	conta := contaDoAdiantamento(p, a.Tipo)
	if conta == nil {
		_, faltando := contabilizacao.Adiantamento(a.Tipo == entity.AdiantamentoTipoPagar, &a.ContaBancariaID, nil, valor, "")
		return nil, faltaConta("dos adiantamentos", faltando)
	}
	var (
		ls       []contabilizacao.Lancamento
		faltando []string
	)
	switch {
	case cp != nil:
		partes, err := partesPagamento(ctx, c, p, cp)
		if err != nil {
			return nil, err
		}
		ls, faltando = contabilizacao.Pagamento(contabilizacao.Baixa{
			Banco: conta, Partes: partes, Principal: valor,
			Historico: fmt.Sprintf("Adiantamento %d aplicado no título %s", a.ID, cp.NumeroDocumento),
		})
	case cr != nil:
		partes, err := partesRecebimento(ctx, c, p, cr)
		if err != nil {
			return nil, err
		}
		doc := ""
		if cr.NumeroDocumento != nil {
			doc = *cr.NumeroDocumento
		}
		ls, faltando = contabilizacao.Recebimento(contabilizacao.Baixa{
			Banco: conta, Partes: partes, Principal: valor,
			Historico: fmt.Sprintf("Adiantamento %d aplicado no título %s", a.ID, doc),
		})
	default:
		return nil, nil
	}
	if len(faltando) > 0 {
		return nil, faltaConta("dos adiantamentos", faltando)
	}
	if err := contasValidas(ctx, c, p.PlanID, ls); err != nil {
		return nil, err
	}
	return &contabilizacao.Lote{PlanID: p.PlanID, Data: data, Prefixo: "ADA", SourceType: contabilizacao.OrigemAdiantamentoAplicacao, Lancamentos: ls}, nil
}
