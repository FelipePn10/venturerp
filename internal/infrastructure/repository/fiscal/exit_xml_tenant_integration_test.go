//go:build integration

package fiscal_test

import (
	"context"
	"testing"
	"time"

	repository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O XML guardado da NF-e é da empresa da nota: a B não lê nem sobrescreve o da A.
func TestExitXMLIsolaEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	ator := testutil.Actor(t, pool)
	ctx := context.Background()
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {})
	var id int64
	if err := pool.QueryRow(ctx, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,status)
		VALUES ($1,$2::date,'5102','Venda',$3,$4,'AUTHORIZED') RETURNING id`,
		testutil.UniqueCode()%1_000_000, time.Now(), ator, a.ID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `DELETE FROM fiscal_exits WHERE id=$1`, id) })
	repo := repository.NewFiscalRepositoryPG(pool)

	if x, _, err := repo.ExitXML(a.Ctx, id); err != nil || x != "" {
		t.Fatalf("nota sem cópia: %q %v", x, err)
	}
	if err := repo.SaveExitXML(a.Ctx, id, "<nfeProc>A</nfeProc>"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.ExitXML(b.Ctx, id); err == nil {
		t.Fatal("empresa B leu o XML da A")
	}
	if err := repo.SaveExitXML(b.Ctx, id, "<nfeProc>B</nfeProc>"); err == nil {
		t.Fatal("empresa B gravou XML na nota da A")
	}
	if err := repo.SaveExitCancelXML(a.Ctx, id, "<procEventoNFe>A</procEventoNFe>"); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveExitCancelXML(b.Ctx, id, "<procEventoNFe>B</procEventoNFe>"); err == nil {
		t.Fatal("empresa B gravou o XML do cancelamento na nota da A")
	}
	if x, c, err := repo.ExitXML(a.Ctx, id); err != nil || x != "<nfeProc>A</nfeProc>" || c != "<procEventoNFe>A</procEventoNFe>" {
		t.Fatalf("XML da A: %q %q %v", x, c, err)
	}
}
