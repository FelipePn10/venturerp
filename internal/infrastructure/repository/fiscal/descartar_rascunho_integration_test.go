//go:build integration

package fiscal_test

import (
	"context"
	"testing"
	"time"

	repository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O descarte da NF-e em rascunho é da empresa da nota: a empresa B não descarta
// a nota da A pelo id, e a autorizada não é descartada.
func TestDescartarRascunhoIsolaEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	ator := testutil.Actor(t, pool)
	ctx := context.Background()
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {})
	nota := func(status string) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,status,chave_acesso)
			VALUES ($1,$2::date,'5102','Venda',$3,$4,$5,'41000000000000000000000000000000000000000000') RETURNING id`,
			testutil.UniqueCode()%1_000_000, time.Now(), ator, a.ID, status).Scan(&id); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { pool.Exec(ctx, `DELETE FROM fiscal_exits WHERE id=$1`, id) })
		return id
	}
	situacao := func(id int64) string {
		var s string
		if err := pool.QueryRow(ctx, `SELECT status || '|' || COALESCE(chave_acesso,'') FROM fiscal_exits WHERE id=$1`, id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	repo := repository.NewFiscalRepositoryPG(pool).(*repository.FiscalRepositoryPG)
	rascunho, autorizada := nota("DRAFT"), nota("AUTHORIZED")
	antes := situacao(rascunho)

	if ok, err := repo.DescartarRascunho(b.Ctx, rascunho, "espião", ator); err != nil || ok {
		t.Fatalf("a empresa B descartou a nota da A: ok=%v err=%v", ok, err)
	}
	if situacao(rascunho) != antes {
		t.Fatalf("a nota da A mudou pela empresa B: %s", situacao(rascunho))
	}
	if ok, err := repo.DescartarRascunho(a.Ctx, autorizada, "teste", ator); err != nil || ok {
		t.Fatalf("nota autorizada foi descartada: ok=%v err=%v", ok, err)
	}
	if ok, err := repo.DescartarRascunho(a.Ctx, rascunho, "teste", ator); err != nil || !ok {
		t.Fatalf("a empresa A não descartou o próprio rascunho: ok=%v err=%v", ok, err)
	}
	if got := situacao(rascunho); got != "CANCELLED|" {
		t.Fatalf("rascunho descartado = %s, quer cancelado e sem chave", got)
	}
}
