//go:build integration

package aps_test

import (
	"context"
	"testing"

	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	apsrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aps"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O parâmetro do sequenciamento é da empresa: sem linha gravada vale o padrão
// da seleção de recursos (só ativos), e o que a empresa A grava a B não lê.
func TestParametroDoSequenciamentoPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		pool.Exec(base, `DELETE FROM manufacturing_sequencing_settings WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
	})
	repo, ok := apsrepo.New(sqlc.New(pool), pool).(domainrepo.ConfigurationRepository)
	if !ok {
		t.Fatal("o repositório do APS deixou de implementar a configuração do sequenciamento")
	}

	for _, e := range []testutil.Empresa{a, b} {
		ativo, err := repo.SequencingSettings(e.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !ativo {
			t.Fatalf("empresa %d sem parâmetro gravado deveria listar só recursos ativos", e.Code)
		}
	}

	if err := repo.UpdateSequencingSettings(a.Ctx, false); err != nil {
		t.Fatal(err)
	}
	if ativo, err := repo.SequencingSettings(a.Ctx); err != nil || ativo {
		t.Fatalf("empresa A gravou 'todos os recursos' e leu ativo=%v (err %v)", ativo, err)
	}
	if ativo, err := repo.SequencingSettings(b.Ctx); err != nil || !ativo {
		t.Fatalf("o parâmetro da empresa A vazou para a B: ativo=%v (err %v)", ativo, err)
	}
}
