//go:build integration

package sales_division_test

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_division"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A conferência de vínculos citava colunas e uma tabela que não existem: excluir
// uma divisão de vendas sempre dava erro.
func TestConferirVinculosDaDivisao(t *testing.T) {
	q, pool := testutil.Queries(t)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {})
	repo := sales_division.NewSalesDivisionRepositorySQLC(q, pool)
	if _, err := repo.HasReferences(a.Ctx, 987654321); err != nil {
		t.Fatalf("conferir vínculos: %v", err)
	}
}
