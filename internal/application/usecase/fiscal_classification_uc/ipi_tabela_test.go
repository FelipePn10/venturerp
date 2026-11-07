package fiscal_classification_uc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
)

type tabelaFixa map[string]decimal.Decimal

func (t tabelaFixa) AliquotaIPIDoNCM(_ context.Context, ncm string) (decimal.Decimal, bool, error) {
	v, ok := t[ncm]
	return v, ok, nil
}

// O pedido de compra sugere o IPI da tabela do NCM, a mesma da nota; a alíquota
// da classificação só cobre NCM fora da tabela.
func TestIPIDaTabelaVenceOdaClassificacao(t *testing.T) {
	uc := &FiscalClassificationUseCase{TabelaNCM: tabelaFixa{"87169090": decimal.RequireFromString("3.25")}}
	if v, ok, _ := uc.TabelaNCM.AliquotaIPIDoNCM(context.Background(), "87169090"); !ok || !v.Equal(decimal.RequireFromString("3.25")) {
		t.Fatalf("tabela: %s %v", v, ok)
	}
}
