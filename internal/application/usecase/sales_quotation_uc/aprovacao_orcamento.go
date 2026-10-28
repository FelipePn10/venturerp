package sales_quotation_uc

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/aprovacao"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository"
)

// Aprovacoes é o motor de aprovação visto pelo orçamento.
type Aprovacoes interface {
	Solicitar(ctx context.Context, t aprovacao.Tipo, documento int64, valor decimal.Decimal, descricao string) (*aprovacao.Solicitacao, error)
	PendenteDoDocumento(ctx context.Context, t aprovacao.Tipo, documento int64) (*aprovacao.Solicitacao, error)
}

// LiberacaoDeOrcamento é o efetivador da aprovação ORCAMENTO_POLITICA_COMERCIAL:
// aprovado, o orçamento é liberado com o número da aprovação no histórico.
type LiberacaoDeOrcamento struct {
	Repo repository.SalesQuotationRepository
}

func (l *LiberacaoDeOrcamento) Efetivar(ctx context.Context, code, solicitacaoID int64) error {
	return l.Repo.ChangeRelease(ctx, code, entity.SalesQuotationReleaseOK, fmt.Sprintf("Liberado pela aprovação nº %d", solicitacaoID))
}

// Reprovar mantém o orçamento bloqueado; o motivo fica no histórico da aprovação.
func (l *LiberacaoDeOrcamento) Reprovar(context.Context, int64, int64, string) error { return nil }
