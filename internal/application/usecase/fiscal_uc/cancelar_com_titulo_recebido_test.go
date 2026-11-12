package fiscal_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	finentity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	financialRepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type titulosDaNota struct {
	financialRepo.FinancialRepository
	titulos []*finentity.ContaReceber
	filtro  financialRepo.CRFilter
}

func (t *titulosDaNota) ListContasReceber(_ context.Context, f financialRepo.CRFilter) ([]*finentity.ContaReceber, error) {
	t.filtro = f
	return t.titulos, nil
}

// Nota com título recebido não é cancelada: o dinheiro ficaria no caixa sem
// venda. Sem recebimento, o cancelamento segue.
func TestRecusarNotaComTituloRecebido(t *testing.T) {
	doc := "NF-1/1"
	aberto := &finentity.ContaReceber{ID: 2, ValorBruto: decimal.NewFromInt(2500)}
	recebido := &finentity.ContaReceber{ID: 1, NumeroDocumento: &doc, ValorBruto: decimal.NewFromInt(2500), ValorRecebido: decimal.NewFromInt(2500)}

	repo := &titulosDaNota{titulos: []*finentity.ContaReceber{aberto}}
	if err := recusarNotaComTituloRecebido(context.Background(), repo, 763); err != nil {
		t.Fatalf("nota sem recebimento deveria poder ser cancelada: %v", err)
	}
	if repo.filtro.FiscalExitID == nil || *repo.filtro.FiscalExitID != 763 {
		t.Fatalf("a consulta não filtrou pela nota: %+v", repo.filtro)
	}

	repo = &titulosDaNota{titulos: []*finentity.ContaReceber{aberto, recebido}}
	err := recusarNotaComTituloRecebido(context.Background(), repo, 763)
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("erro = %v, esperado validação", err)
	}
	for _, trecho := range []string{"NF-1/1", "2500.00", "estorne o recebimento"} {
		if !strings.Contains(err.Error(), trecho) {
			t.Fatalf("%q não contém %q", err.Error(), trecho)
		}
	}
}
