//go:build integration

package customer_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	customerpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O apetite de crédito da empresa sai do faturamento dela: notas de venda
// autorizadas dos últimos 12 meses, só da própria empresa, contando os meses
// em que houve nota. Rascunho, nota antiga, nota de outra empresa, devolução e
// bonificação ficam fora.
func TestSellerVolume_FaturamentoDaEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		testutil.Exec(t, pool, `DELETE FROM fiscal_exits WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
		testutil.Exec(t, pool, `DELETE FROM invoice_types WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
	})
	actor := testutil.Actor(t, pool)
	life := customerpg.NewLifecycle(customerpg.New(sqlc.New(pool), pool))
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	nota := func(empresa int64, dia time.Time, valor, status string) {
		testutil.Exec(t, pool, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,status,valor_total)
			VALUES(1,$1::date,'5101','VENDA',$2,$3,$4,$5::numeric)`, dia.Format("2006-01-02"), actor, empresa, status, valor)
	}
	nota(a.ID, today, "1000", "AUTHORIZED")
	nota(a.ID, today.AddDate(0, -1, 0), "2000", "AUTHORIZED")
	nota(a.ID, today.AddDate(0, -1, 0), "500", "AUTHORIZED")
	nota(a.ID, today, "9999", "DRAFT")
	nota(a.ID, today.AddDate(0, -13, 0), "9999", "AUTHORIZED")
	nota(b.ID, today, "7777", "AUTHORIZED")
	// Devolução (finalidade 4) e remessa de tipo que não gera receita não são venda.
	testutil.Exec(t, pool, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,status,valor_total,finalidade)
		VALUES(1,$1::date,'5202','DEVOLUCAO',$2,$3,'AUTHORIZED',8888,4)`, today.Format("2006-01-02"), actor, a.ID)
	var bonif int64
	if err := pool.QueryRow(a.Ctx, `INSERT INTO invoice_types(code,description,enterprise_id,generates_revenue) VALUES(91,'BONIFICAÇÃO',$1,false) RETURNING id`, a.ID).Scan(&bonif); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,status,valor_total,invoice_type_id)
		VALUES(1,$1::date,'5910','BONIFICACAO',$2,$3,'AUTHORIZED',6666,$4)`, today.Format("2006-01-02"), actor, a.ID, bonif)

	v, err := life.SellerVolume(a.Ctx, today)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Billed12m.Equal(decimal.NewFromInt(3500)) || v.Months != 2 {
		t.Fatalf("faturamento %s em %d meses, quer 3500 em 2", v.Billed12m, v.Months)
	}
	if !v.MonthlyAverage().Equal(decimal.NewFromInt(1750)) {
		t.Fatalf("média mensal %s", v.MonthlyAverage())
	}
}
