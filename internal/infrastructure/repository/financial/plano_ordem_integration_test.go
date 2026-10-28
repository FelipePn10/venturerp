//go:build integration

package financial_test

import (
	"fmt"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	financialrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O plano de contas sai em ordem natural: 2.1.10 depois de 2.1.9 (e não logo
// abaixo de 2.1.1, como na ordem de texto), e o pai antes dos filhos.
func TestIntegration_PlanoDeContasEmOrdemNatural(t *testing.T) {
	pool := testutil.Pool(t)
	repo := financialrepo.NewFinancialRepositoryPG(pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		testutil.Exec(t, pool, `DELETE FROM plano_contas WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
	})
	for _, c := range []string{"2.1.10", "2.1.2", "2.10", "2", "2.1.1", "2.1", "2.1.9", "3", "2.2"} {
		testutil.Exec(t, pool, `INSERT INTO plano_contas(codigo,descricao,tipo,natureza,nivel,enterprise_id) VALUES($1,$1,'DESPESA','DEBITO',1,$2)`, c, a.ID)
	}
	quer := "[2 2.1 2.1.1 2.1.2 2.1.9 2.1.10 2.2 2.10 3]"
	ativos, err := repo.ListPlanoContas(a.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	todos, err := repo.ListPlanoContasComInativos(a.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		nome  string
		lista []*entity.PlanoContas
	}{{"ativos", ativos}, {"com inativos", todos}} {
		var got []string
		for _, p := range c.lista {
			got = append(got, p.Codigo)
		}
		if fmt.Sprint(got) != quer {
			t.Fatalf("%s: ordem = %v, quer %s", c.nome, got, quer)
		}
	}
}
