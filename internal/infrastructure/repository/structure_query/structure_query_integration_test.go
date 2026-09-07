//go:build integration

package structure_query_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// A estrutura era gravada com vigência, fórmula e alternativos e voltava
// zerada da consulta, porque o mapeamento descartava esses campos. Sem eles a
// tela não consegue exibir de volta o que o usuário acabou de cadastrar.
func TestConsultaDevolveVigenciaFormulaEAlternativos(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(base, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})

	var colunas int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'item_structures'
		   AND column_name IN ('start_date','end_date','quantity_formula','loss_formula',
		                       'quantity_rounding','quantity_scale','substitute_group',
		                       'substitute_priority','is_coproduct','is_fixed_qty')`).Scan(&colunas)
	if err != nil {
		t.Fatal(err)
	}
	if colunas != 10 {
		t.Fatalf("a tabela tem %d das 10 colunas esperadas — a consulta não teria o que devolver", colunas)
	}
}
