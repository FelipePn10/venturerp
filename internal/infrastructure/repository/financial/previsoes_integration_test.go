//go:build integration

package financial_test

import (
	"context"
	"testing"
	"time"

	financialpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O fluxo projetado é da empresa: o título e o pedido da A não aparecem na B.
func TestPrevisoesDeCaixaPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		pool.Exec(bg, `DELETE FROM contas_receber WHERE enterprise_id=$1`, a.ID)
	})
	hoje := time.Now()
	testutil.Exec(t, pool, `INSERT INTO contas_receber(numero_documento,data_emissao,data_vencimento,valor_bruto,valor_recebido,status,criado_por,enterprise_id)
		VALUES('NF 1/1',$3::date,$3::date + 10,500,0,'PENDENTE',$1,$2)`, ator, a.ID, hoje)
	repo := financialpg.NewFinancialRepositoryPG(pool).(*financialpg.FinancialRepositoryPG)
	pa, err := repo.PrevisoesDeCaixa(a.Ctx, hoje.AddDate(0, 0, 30))
	if err != nil || len(pa) != 1 || pa[0].Origem != "RECEBER_TITULO" {
		t.Fatalf("empresa A: %+v err=%v", pa, err)
	}
	if pb, err := repo.PrevisoesDeCaixa(b.Ctx, hoje.AddDate(0, 0, 30)); err != nil || len(pb) != 0 {
		t.Fatalf("empresa B viu o título da A: %+v err=%v", pb, err)
	}
}

// A DRE gerencial lê só o plano da empresa: a B, informando o plano da A, não vê contas.
func TestDREGerencialPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	a, b := testutil.DuasEmpresas(t, pool, nil)
	var plano int64
	if err := pool.QueryRow(bg, `INSERT INTO accounting_plans(plan_number,description,valid_from,status,enterprise_id) VALUES($1,'DRE','2020-01-01','A',$2) RETURNING id`,
		testutil.UniqueCode()%1_000_000+600_000, a.ID).Scan(&plano); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO accounting_accounts(plan_id,account_number,description,nature_code,valid_from,is_analytic) VALUES($1,'3.1','Vendas','C','2020-01-01',TRUE)`, plano)
	t.Cleanup(func() {
		pool.Exec(bg, `DELETE FROM accounting_accounts WHERE plan_id=$1`, plano)
		pool.Exec(bg, `DELETE FROM accounting_plans WHERE id=$1`, plano)
	})
	repo := financialpg.NewFinancialRepositoryPG(pool).(*financialpg.FinancialRepositoryPG)
	hoje := time.Now()
	if ca, err := repo.ContasComMovimento(a.Ctx, &plano, hoje.AddDate(0, -1, 0), hoje); err != nil || len(ca) != 1 {
		t.Fatalf("empresa A: %+v err=%v", ca, err)
	}
	if cb, err := repo.ContasComMovimento(b.Ctx, &plano, hoje.AddDate(0, -1, 0), hoje); err != nil || len(cb) != 0 {
		t.Fatalf("empresa B leu o plano da A: %+v err=%v", cb, err)
	}
}
