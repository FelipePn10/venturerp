//go:build integration

package fiscal_uc

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	financialpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// Adiantamento contabilizado (VFIN0600), numa empresa própria:
//
//  1. contabilização ligada sem a conta de adiantamentos: o lançamento é recusado
//     dizendo o que falta, e nada se move no banco;
//  2. pago a fornecedor 1.000: D Adiantamentos a fornecedores / C Banco;
//  3. aplicado 400 num título avulso: D despesa do plano / C Adiantamentos;
//  4. outro adiantamento de 200 cancelado: a entrada é estornada;
//  5. recebido de cliente 300: D Banco / C Adiantamentos de clientes.
//
// Ao fim o balancete fecha e cada conta tem o saldo esperado.
func TestIntegration_AdiantamentoContabilizado(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	u := testutil.UniqueCode()
	actor := uuid.New()
	code := int64(1_500_000_000 + u%90_000_000)
	var empresa int64
	if _, err := pool.Exec(bg, `INSERT INTO users(id,name,email,password) VALUES($1,'Adiant',$2,'x')`, actor, actor.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Adiantamento contábil') RETURNING id`, code).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: empresa, EnterpriseCode: code})
	hoje := time.Now().Format("2006-01-02")

	var banco int64
	if err := pool.QueryRow(bg, `INSERT INTO contas_bancarias(banco,agencia,conta,descricao,saldo_inicial,is_active,created_by,enterprise_id) VALUES('001','1','1','Banco','10000',TRUE,$1,$2) RETURNING id`,
		actor, empresa).Scan(&banco); err != nil {
		t.Fatal(err)
	}
	var plano int64
	if err := pool.QueryRow(bg, `INSERT INTO accounting_plans(plan_number,description,valid_from,status,enterprise_id) VALUES($1,'Adiant','2020-01-01','A',$2) RETURNING id`, u%1_000_000+600_000, empresa).Scan(&plano); err != nil {
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
	cBanco, cForn, cAdF, cAdC, cDesp, cClientes := conta("1.1.1 BANCO"), conta("2.1.1 FORNECEDORES"), conta("1.1.9 ADIANT FORN"), conta("2.1.9 ADIANT CLI"), conta("3.1 SERVICOS"), conta("1.1.2 CLIENTES")
	var planoDesp int64
	if err := pool.QueryRow(bg, `INSERT INTO plano_contas(codigo,descricao,tipo,natureza,enterprise_id,accounting_account_id) VALUES($1,'Serviços','A','D',$2,$3) RETURNING id`,
		fmt.Sprintf("A%d.SERV", u), empresa, cDesp).Scan(&planoDesp); err != nil {
		t.Fatal(err)
	}
	fornecedor, cliente := int64(8_000_000_000+u%1_000_000_000), int64(9_000_000_000+u%1_000_000_000)
	var titulo int64
	if err := pool.QueryRow(bg, `INSERT INTO contas_pagar(numero_documento,fornecedor_id,data_emissao,data_vencimento,valor_bruto,valor_pago,valor_adiantamento_abatido,plano_contas_id,criado_por,enterprise_id)
		VALUES('AD-1',$1,$2::date,$2::date,1000,0,0,$3,$4,$5) RETURNING id`, fornecedor, hoje, planoDesp, actor, empresa).Scan(&titulo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM accounting_journal_entries WHERE empresa_id=$1`, `DELETE FROM accounting_posting_params WHERE enterprise_id=$1`,
			`DELETE FROM adiantamento_aplicacoes WHERE adiantamento_id IN (SELECT id FROM adiantamentos WHERE enterprise_id=$1)`,
			`DELETE FROM adiantamentos WHERE enterprise_id=$1`, `DELETE FROM fluxo_caixa WHERE enterprise_id=$1`,
			`DELETE FROM contas_pagar WHERE enterprise_id=$1`, `DELETE FROM contas_bancarias WHERE enterprise_id=$1`,
			`DELETE FROM plano_contas WHERE enterprise_id=$1`,
		} {
			if _, err := pool.Exec(bg, q, empresa); err != nil {
				t.Logf("limpeza (%s): %v", q, err)
			}
		}
		_, _ = pool.Exec(bg, `DELETE FROM accounting_accounts WHERE plan_id=$1`, plano)
		_, _ = pool.Exec(bg, `DELETE FROM accounting_plans WHERE id=$1`, plano)
		if _, err := pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, empresa); err != nil {
			t.Errorf("a limpeza deixou dados: %v", err)
		}
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id=$1`, actor)
	})

	auth := &infraauth.AuthService{}
	fiscalRepo := fiscalpg.NewFiscalRepositoryPG(pool)
	docs := fiscalpg.NewFiscalEntryDocumentRepositoryPG(fiscalRepo)
	finRepo := financialpg.NewFinancialRepositoryPG(pool)
	pg := fiscalRepo.(*fiscalpg.FiscalRepositoryPG)
	contab := &AccountingParamsUseCase{Docs: docs, Auth: auth}
	params := repository.AccountingPostingParams{
		PlanID: plano, FornecedoresAccountID: cForn, ContabilizarPagamentos: true, ContabilizarRecebimentos: true,
		ClientesAccountID: &cClientes,
	}
	if _, err := contab.Save(ctx, params); err != nil {
		t.Fatalf("parâmetros: %v", err)
	}
	if err := contab.VincularContaBancaria(ctx, banco, &cBanco); err != nil {
		t.Fatalf("vínculo do banco: %v", err)
	}

	criar := &financial_uc.CreateAdiantamentoUseCase{Repo: finRepo, Auth: auth, Contabil: pg}
	aplicar := &financial_uc.AplicarAdiantamentoUseCase{Repo: finRepo, Auth: auth, Contabil: pg}
	cancelar := &financial_uc.CancelarAdiantamentoUseCase{Repo: finRepo, Auth: auth}
	pagar := func(valor float64) request.CreateAdiantamentoDTO {
		return request.CreateAdiantamentoDTO{Tipo: "PAGAR", ParceiroID: &fornecedor, ContaBancariaID: banco, DataAdiantamento: hoje, ValorOriginal: valor}
	}
	saldoBanco := func() decimal.Decimal {
		var s decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT saldo_inicial FROM contas_bancarias WHERE id=$1`, banco).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	// 1. Sem a conta de adiantamentos: recusado, sem mexer no banco.
	if _, err := criar.Execute(ctx, pagar(1000)); err == nil || !strings.Contains(err.Error(), "adiantamentos a fornecedores") {
		t.Fatalf("sem a conta de adiantamentos o lançamento deveria ser recusado dizendo o que falta, veio %v", err)
	}
	if !saldoBanco().Equal(decimal.NewFromInt(10000)) {
		t.Fatalf("o banco não pode mexer quando a contabilização recusa: %s", saldoBanco())
	}
	params.AdiantamentoFornecedoresAccountID, params.AdiantamentoClientesAccountID = &cAdF, &cAdC
	if _, err := contab.Save(ctx, params); err != nil {
		t.Fatalf("parâmetros com adiantamentos: %v", err)
	}

	// 2. Pago a fornecedor.
	adv, err := criar.Execute(ctx, pagar(1000))
	if err != nil {
		t.Fatalf("adiantamento: %v", err)
	}
	// 3. Aplicado 400 no título avulso.
	if _, err := aplicar.Execute(ctx, adv.ID, request.AplicarAdiantamentoDTO{ContaTipo: "PAGAR", ContaID: titulo, Valor: 400, DataAplicacao: &hoje}); err != nil {
		t.Fatalf("aplicação: %v", err)
	}
	// 4. Outro, cancelado.
	outro, err := criar.Execute(ctx, pagar(200))
	if err != nil {
		t.Fatalf("segundo adiantamento: %v", err)
	}
	if _, err := cancelar.Execute(ctx, outro.ID, "lançado em dobro"); err != nil {
		t.Fatalf("cancelamento: %v", err)
	}
	// 5. Recebido de cliente.
	if _, err := criar.Execute(ctx, request.CreateAdiantamentoDTO{Tipo: "RECEBER", ParceiroID: &cliente, ContaBancariaID: banco, DataAdiantamento: hoje, ValorOriginal: 300}); err != nil {
		t.Fatalf("adiantamento de cliente: %v", err)
	}

	rows, err := pool.Query(bg, `SELECT conta, SUM(v) FROM (
		SELECT debit_account_id conta, value v FROM accounting_journal_entries WHERE empresa_id=$1
		UNION ALL SELECT credit_account_id, -value FROM accounting_journal_entries WHERE empresa_id=$1) x GROUP BY conta`, empresa)
	if err != nil {
		t.Fatal(err)
	}
	saldos := map[int64]decimal.Decimal{}
	total := decimal.Zero
	for rows.Next() {
		var c int64
		var v decimal.Decimal
		if err := rows.Scan(&c, &v); err != nil {
			t.Fatal(err)
		}
		saldos[c] = v
		total = total.Add(v)
	}
	rows.Close()
	if !total.IsZero() {
		t.Fatalf("o balancete não fecha: %s", total)
	}
	esperado := map[int64]int64{cBanco: -1000 + 300, cAdF: 600, cDesp: 400, cAdC: -300}
	for c, v := range esperado {
		if !saldos[c].Equal(decimal.NewFromInt(v)) {
			t.Errorf("conta %d: saldo %s, esperado %d (saldos %v)", c, saldos[c], v, saldos)
		}
	}
	if !saldoBanco().Equal(decimal.NewFromInt(10000 - 1000 + 300)) {
		t.Errorf("saldo do banco = %s, esperado 9300", saldoBanco())
	}
}
