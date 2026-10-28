//go:build integration

package financial_uc_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	financialrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A apuração do mês recolhe o ICMS próprio DESTACADO (operação − diferido),
// e o ICMS-ST, o DIFAL e o FCP por estado de destino, cada um com seu título.
// A outra empresa não enxerga nada disso.
func TestApuracaoComSTEDIFALPorUF(t *testing.T) {
	pool := testutil.Pool(t)
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []int64{a.ID, b.ID} {
			testutil.Exec(t, pool, `DELETE FROM tax_assessments WHERE enterprise_id=$1`, e)
			testutil.Exec(t, pool, `DELETE FROM contas_pagar WHERE enterprise_id=$1`, e)
			testutil.Exec(t, pool, `DELETE FROM fiscal_exit_items WHERE fiscal_exit_id IN (SELECT id FROM fiscal_exits WHERE enterprise_id=$1)`, e)
			testutil.Exec(t, pool, `DELETE FROM fiscal_exits WHERE enterprise_id=$1`, e)
			testutil.Exec(t, pool, `DELETE FROM fiscal_configs WHERE enterprise_id=$1`, e)
		}
	})
	for _, e := range []testutil.Empresa{a, b} {
		testutil.Exec(t, pool, `INSERT INTO fiscal_configs(cnpj_empresa,razao_social,uf_empresa,updated_by,enterprise_id) VALUES('11222333000181','EMP','PR',$1,$2)`, ator, e.ID)
	}
	hoje := time.Now()
	competencia := hoje.Format("01/2006")
	nota := func(uf string, icmsCab float64) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(a.Ctx, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,status,uf_destinatario,valor_icms,valor_icms_st)
			VALUES($1,$2::date,'5102','Venda',$3,$4,'AUTHORIZED',$5,$6,0) RETURNING id`, testutil.UniqueCode()%1_000_000, hoje, ator, a.ID, uf, icmsCab).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	linha := func(nota int64, icms, dif, st, difal, fcp float64) {
		t.Helper()
		testutil.Exec(t, pool, `INSERT INTO fiscal_exit_items(fiscal_exit_id,sequence,cfop,quantity,unit_price,total_price,valor_icms,valor_icms_diferido,valor_icms_st,icms_uf_dest_valor,fcp_uf_dest_valor)
			VALUES($1,1,'5102',1,1000,1000,$2,$3,$4,$5,$6)`, nota, icms, dif, st, difal, fcp)
	}
	// Revenda no PR com diferimento: cabeçalho já com o ICMS destacado (120).
	pr := nota("PR", 120)
	linha(pr, 195, 75, 0, 0, 0)
	// ST para o PR e para SP; DIFAL para GO; DIFAL + FCP para o RJ.
	st := nota("PR", 195)
	linha(st, 195, 0, 91.65, 0, 0)
	sp := nota("SP", 120)
	linha(sp, 120, 0, 163.96, 0, 0)
	goNota := nota("GO", 73.5)
	linha(goNota, 73.5, 0, 0, 126, 0)
	rj := nota("RJ", 126)
	linha(rj, 126, 0, 0, 84, 21)

	uc := &financial_uc.ApurarImpostosUseCase{Repo: financialrepo.NewFinancialRepositoryPG(pool), Auth: &infraauth.AuthService{}}
	out, err := uc.Execute(a.Ctx, request.ApurarImpostosDTO{Competencia: competencia})
	if err != nil {
		t.Fatal(err)
	}
	quer := map[string]float64{"ICMS": 120 + 195 + 120 + 73.5 + 126, "ST-PR": 91.65, "ST-SP": 163.96, "DIFAL-GO": 126, "DIFAL-RJ": 84, "FCP-RJ": 21}
	achou := map[string]bool{}
	for _, r := range out {
		v, ok := quer[r.Imposto]
		if !ok {
			continue
		}
		achou[r.Imposto] = true
		if !r.SaldoDevedor.Equal(decimal.NewFromFloat(v)) || r.CpID == nil {
			t.Errorf("%s: saldo %s título %v, quer %.2f com título", r.Imposto, r.SaldoDevedor, r.CpID, v)
		}
	}
	for imp := range quer {
		if !achou[imp] {
			t.Errorf("apuração sem %s", imp)
		}
	}

	// A outra empresa não tem nada a apurar desta.
	outB, err := uc.Execute(b.Ctx, request.ApurarImpostosDTO{Competencia: competencia})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range outB {
		if !r.SaldoDevedor.IsZero() {
			t.Fatalf("empresa B apurou %s = %s da empresa A", r.Imposto, r.SaldoDevedor)
		}
	}
}
