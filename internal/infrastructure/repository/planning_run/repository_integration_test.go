//go:build integration

package planning_run_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planning_run"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O cadeado precisa impedir de fato uma segunda execução, e precisa ser
// liberável. Como o advisory lock pertence à conexão que o tomou e o pool não
// garante devolver a mesma, este teste roda contra o Postgres real.
func TestAdvisoryLockBloqueiaSegundaExecucaoELibera(t *testing.T) {
	pool := testutil.Pool(t)
	repo := planning_run.New(pool)
	ctx := context.Background()
	const empresa int64 = 987654

	ok, err := repo.TryLock(ctx, empresa)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("não conseguiu o cadeado inicial")
	}

	// Um segundo repositório simula outro processo da API.
	outro := planning_run.New(pool)
	ok2, err := outro.TryLock(ctx, empresa)
	if err != nil {
		t.Fatal(err)
	}
	if ok2 {
		_ = outro.Unlock(ctx, empresa)
		t.Fatal("segunda execução conseguiu o cadeado enquanto a primeira o detinha")
	}

	if err := repo.Unlock(ctx, empresa); err != nil {
		t.Fatal(err)
	}

	// Depois de liberado, a próxima execução entra.
	ok3, err := outro.TryLock(ctx, empresa)
	if err != nil {
		t.Fatal(err)
	}
	if !ok3 {
		t.Fatal("cadeado não foi liberado — a empresa ficaria travada")
	}
	if err := outro.Unlock(ctx, empresa); err != nil {
		t.Fatal(err)
	}
}
