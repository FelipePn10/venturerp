//go:build integration

package fiscal_uc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	financialpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// IBS e CBS na contabilização da venda e na apuração mensal, numa empresa
// própria (código ≠ id):
//
//   - a NF-e de dezembro/2026 traz IBS/CBS destacados, mas é ano de teste: nada
//     vai para "a recolher";
//   - a de janeiro/2027 lança D IBS/CBS sobre vendas × C a recolher, sem mudar
//     Clientes nem a receita bruta;
//   - a apuração de 01/2027 soma o débito da saída e o crédito da entrada que
//     gera crédito, e ignora o item de entrada sem direito a crédito.
func TestIntegration_ReformaContabilizacaoEApuracao(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	u := testutil.UniqueCode()
	actor := uuid.New()
	code := int64(1_500_000_000 + u%90_000_000)
	var empresa int64
	if _, err := pool.Exec(bg, `INSERT INTO users(id,name,email,password) VALUES($1,'Reforma',$2,'x')`, actor, actor.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Reforma contábil') RETURNING id`, code).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: empresa, EnterpriseCode: code})

	var plano int64
	if err := pool.QueryRow(bg, `INSERT INTO accounting_plans(plan_number,description,valid_from,status,enterprise_id) VALUES($1,'Reforma','2020-01-01','A',$2) RETURNING id`, u%1_000_000+600_000, empresa).Scan(&plano); err != nil {
		t.Fatal(err)
	}
	conta := func(numero string) int64 {
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO accounting_accounts(plan_id,account_number,description,nature_code,valid_from,is_analytic) VALUES($1,$2,$2,'D','2020-01-01',TRUE) RETURNING id`,
			plano, numero).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	cForn, cClientes, cReceita := conta("2.1.1 FORN"), conta("1.1.2 CLIENTES"), conta("4.1 RECEITA")
	cIBSv, cIBSr, cCBSv, cCBSr := conta("3.6 IBS VENDAS"), conta("2.1.8 IBS RECOLHER"), conta("3.7 CBS VENDAS"), conta("2.1.9 CBS RECOLHER")

	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM accounting_journal_entries WHERE empresa_id=$1`, `DELETE FROM accounting_posting_params WHERE enterprise_id=$1`,
			`DELETE FROM fiscal_entry_items WHERE fiscal_entry_id IN (SELECT id FROM fiscal_entries WHERE enterprise_id=$1)`,
			`DELETE FROM fiscal_entries WHERE enterprise_id=$1`, `DELETE FROM fiscal_exits WHERE enterprise_id=$1`,
		} {
			if _, err := pool.Exec(bg, q, empresa); err != nil {
				t.Logf("limpeza (%s): %v", q, err)
			}
		}
		_, _ = pool.Exec(bg, `DELETE FROM accounting_accounts WHERE plan_id=$1`, plano)
		_, _ = pool.Exec(bg, `DELETE FROM accounting_plans WHERE id=$1`, plano)
		_, _ = pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, empresa)
		_, _ = pool.Exec(bg, `UPDATE users SET is_active=FALSE WHERE id=$1`, actor)
	})

	testutil.Exec(t, pool, `INSERT INTO accounting_posting_params(enterprise_id,plan_id,fornecedores_account_id,contabilizar_saidas,clientes_account_id,receita_vendas_account_id,
		ibs_vendas_account_id,ibs_recolher_account_id,cbs_vendas_account_id,cbs_recolher_account_id,updated_by)
		VALUES($1,$2,$3,TRUE,$4,$5,$6,$7,$8,$9,$10)`, empresa, plano, cForn, cClientes, cReceita, cIBSv, cIBSr, cCBSv, cCBSr, actor)

	pg := fiscalpg.NewFiscalRepositoryPG(pool).(*fiscalpg.FiscalRepositoryPG)
	saida := func(numero int64, emissao time.Time, ibs, cbs string) *entity.FiscalExit {
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO fiscal_exits(numero_nf,serie,data_emissao,cfop,natureza_operacao,valor_produtos,valor_frete,valor_seguro,valor_desconto,
			valor_ipi,valor_icms,valor_pis,valor_cofins,valor_total,status,is_active,created_by,emitida_contingencia,base_icms_st,valor_icms_st,valor_ibs,valor_cbs,enterprise_id)
			VALUES($1,'1',$2::date,'5101','Venda',1000,0,0,0,0,0,0,0,1000,'AUTHORIZED',TRUE,$3,FALSE,0,0,$4::numeric,$5::numeric,$6) RETURNING id`,
			numero, emissao, actor, ibs, cbs, empresa).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return &entity.FiscalExit{ID: id, NumeroNF: numero, Serie: "1", DataEmissao: emissao, ValorProdutos: 1000, ValorTotal: 1000}
	}
	saldo := func() map[int64]decimal.Decimal {
		m := map[int64]decimal.Decimal{}
		rows, err := pool.Query(bg, `SELECT debit_account_id, credit_account_id, value FROM accounting_journal_entries WHERE empresa_id=$1`, empresa)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var d, c int64
			var v decimal.Decimal
			if err := rows.Scan(&d, &c, &v); err != nil {
				t.Fatal(err)
			}
			m[d], m[c] = m[d].Add(v), m[c].Sub(v)
		}
		return m
	}

	// Dezembro/2026: só a receita.
	dez := saida(9101, time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC), "1.00", "9.00")
	if avisos := contabilizarSaida(ctx, pg, dez, true); len(avisos) > 0 {
		t.Fatalf("contabilização de 2026: %v", avisos)
	}
	m := saldo()
	if !m[cIBSr].IsZero() || !m[cCBSr].IsZero() || !m[cReceita].Equal(decimal.RequireFromString("-1000")) {
		t.Fatalf("2026 não recolhe IBS/CBS: IBS %s CBS %s receita %s", m[cIBSr], m[cCBSr], m[cReceita])
	}

	// Janeiro/2027: IBS 0,1% (UF + Mun) e CBS 8,8% sobre 1.000.
	jan := saida(9102, time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC), "1.00", "88.00")
	if avisos := contabilizarSaida(ctx, pg, jan, true); len(avisos) > 0 {
		t.Fatalf("contabilização de 2027: %v", avisos)
	}
	m = saldo()
	casos := map[int64]string{cClientes: "2000", cReceita: "-2000", cIBSv: "1", cIBSr: "-1", cCBSv: "88", cCBSr: "-88"}
	for c, v := range casos {
		if !m[c].Equal(decimal.RequireFromString(v)) {
			t.Fatalf("conta %d = %s, esperado %s", c, m[c], v)
		}
	}
	total := decimal.Zero
	for _, v := range m {
		total = total.Add(v)
	}
	if !total.IsZero() {
		t.Fatalf("balancete não fecha: %s", total)
	}

	// Entrada de janeiro/2027: um item com crédito (IBS 0,50, CBS 44) e outro
	// sem direito a crédito (uso e consumo).
	var entrada int64
	if err := pool.QueryRow(bg, `INSERT INTO fiscal_entries(numero_nf,data_emissao,data_entrada,cnpj_emitente,razao_social_emitente,status,is_active,created_by,enterprise_id)
		VALUES(7101,'2027-01-10','2027-01-12',$1,'FORNECEDOR','APPROVED',TRUE,$2,$3) RETURNING id`, fmt.Sprintf("%014d", u%100_000_000_000_000), actor, empresa).Scan(&entrada); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO fiscal_entry_items(fiscal_entry_id,quantity,unit_price,total_price,cfop,valor_ibs,valor_cbs,gera_credito_ibscbs)
		VALUES($1,1,500,500,'1101',0.50,44,TRUE),($1,1,100,100,'1556',0.10,8.80,FALSE)`, entrada)

	fin := financialpg.NewFinancialRepositoryPG(pool)
	deb, err := fin.GetFiscalDebits(ctx, "01/2027")
	if err != nil {
		t.Fatal(err)
	}
	cred, err := fin.GetFiscalCredits(ctx, "01/2027")
	if err != nil {
		t.Fatal(err)
	}
	if !deb["IBS"].Equal(decimal.RequireFromString("1")) || !deb["CBS"].Equal(decimal.RequireFromString("88")) {
		t.Fatalf("débitos 01/2027: IBS %s CBS %s", deb["IBS"], deb["CBS"])
	}
	if !cred["IBS"].Equal(decimal.RequireFromString("0.5")) || !cred["CBS"].Equal(decimal.RequireFromString("44")) {
		t.Fatalf("créditos 01/2027 (só o item com crédito): IBS %s CBS %s", cred["IBS"], cred["CBS"])
	}
	debDez, err := fin.GetFiscalDebits(ctx, "12/2026")
	if err != nil {
		t.Fatal(err)
	}
	if !debDez["CBS"].Equal(decimal.RequireFromString("9")) {
		t.Fatalf("o débito de 12/2026 é somado (informativo) — quem decide não apurar é o caso de uso: %s", debDez["CBS"])
	}
}
