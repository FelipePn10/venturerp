//go:build integration

package enterprise_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	enterpriserepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/enterprise"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestEnterpriseListOnlyShowsUserEnterprises: a lista de empresas devolvia
// TODAS as empresas do banco para qualquer usuário autenticado.
func TestEnterpriseListOnlyShowsUserEnterprises(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	a, b := testutil.DuasEmpresas(t, pool, nil)
	actor := testutil.Actor(t, pool)
	if _, err := pool.Exec(base, `INSERT INTO user_enterprises(user_id,enterprise_id,role) VALUES($1,$2,'USER')`, actor, a.ID); err != nil {
		t.Fatal(err)
	}
	repo := enterpriserepo.NewRepositoryEnterpriseSQLC(sqlc.New(pool))

	lista, err := repo.ListForUser(base, actor)
	if err != nil {
		t.Fatal(err)
	}
	var viuA bool
	for _, e := range lista {
		if int64(e.ID) == b.ID {
			t.Fatal("a lista trouxe empresa à qual o usuário não pertence")
		}
		viuA = viuA || int64(e.ID) == a.ID
	}
	if !viuA {
		t.Fatal("a lista não trouxe a empresa do usuário")
	}
	if _, err := repo.GetByCodeForUser(base, int(b.Code), actor); err == nil {
		t.Fatal("o usuário leu empresa à qual não pertence")
	}
	if got, err := repo.GetByCodeForUser(base, int(a.Code), actor); err != nil || int64(got.ID) != a.ID {
		t.Fatalf("o usuário não leu a própria empresa: %v", err)
	}
}
