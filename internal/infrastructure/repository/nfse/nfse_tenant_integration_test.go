//go:build integration

package nfse_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/nfse/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/nfse"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestNFSeIsIsolatedByTenant: a NFS-e não tinha empresa — pelo id, uma
// empresa lia, autorizava e cancelava (com o próprio token Focus) a nota da
// outra.
func TestNFSeIsIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(context.Background(), `DELETE FROM nfse WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
	})
	repo := nfse.NewNFSeRepositoryPG(pool)
	nota, err := repo.Create(a.Ctx, &entity.NFSe{TipoRPS: 1, NaturezaOperacao: 1, DataEmissao: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		Status: "RASCUNHO", ItemListaServico: "14.01", Discriminacao: "SERVICO", CodigoMunicipio: "4106902", ValorServicos: 100})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByID(b.Ctx, nota.ID); err == nil {
		t.Fatal("empresa B leu a NFS-e da A")
	}
	if l, err := repo.List(b.Ctx); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d NFS-e da A (%v)", len(l), err)
	}
	if _, err := repo.UpdateStatus(b.Ctx, nota.ID, "CANCELADA"); err == nil {
		t.Fatal("empresa B cancelou a NFS-e da A")
	}
	if _, err := repo.UpdateAuthorization(b.Ctx, nota.ID, "1", "X", "u", "ref"); err == nil {
		t.Fatal("empresa B gravou autorização na NFS-e da A")
	}
	got, err := repo.GetByID(a.Ctx, nota.ID)
	if err != nil || got.Status != "RASCUNHO" {
		t.Fatalf("a NFS-e da A mudou: %+v %v", got, err)
	}
}
