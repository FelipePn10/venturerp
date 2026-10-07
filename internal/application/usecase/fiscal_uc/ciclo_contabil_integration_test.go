//go:build integration

package fiscal_uc

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	financialpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	stockpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// O ciclo contábil inteiro numa empresa própria (código ≠ id):
//
//  1. NF de entrada de 100 mil com IRRF de 1.500, aprovada e contabilizada;
//  2. 1ª duplicata paga em parte, com atraso (juros e multa) e desconto: o saldo
//     vira outro título com a empresa e o rateio, e é pago depois;
//  3. o IRRF retido é recolhido;
//  4. venda: a saída de estoque é valorizada pelo custo médio, a nota é
//     contabilizada (receita, impostos, CMV) e o cliente paga;
//  5. a venda é cancelada: o estoque volta e a contabilidade estorna.
//
// Ao fim o balancete fecha e cada conta tem o saldo esperado.
func TestIntegration_CicloContabil(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	u := testutil.UniqueCode()
	actor := uuid.New()
	code := int64(1_400_000_000 + u%90_000_000)
	var empresa int64
	if _, err := pool.Exec(bg, `INSERT INTO users(id,name,email,password) VALUES($1,'Ciclo',$2,'x')`, actor, actor.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Ciclo contábil') RETURNING id`, code).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: empresa, EnterpriseCode: code})
	cnpj := fmt.Sprintf("%014d", (u+31)%100_000_000_000_000)
	supplier := 7_000_000_000 + u%1_000_000_000
	chapa, luva := u+20, u+21
	chave := fmt.Sprintf("41%042d", u+9)

	testutil.Exec(t, pool, `INSERT INTO fiscal_configs(cnpj_empresa,razao_social,uf_empresa,updated_by,enterprise_id) VALUES('98765432000110','INDUSTRIA','PR',$1,$2)`, actor, empresa)
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,'Almox',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`,
		fmt.Sprintf("WC-%d", u), actor, empresa).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	for _, c := range []int64{chapa, luva} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name) VALUES($1,($1::bigint)::text,$2,$3,$4,'ITEM '||$1::text)`, c, almox, actor, empresa)
	}
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'ACO LTDA',$2,$3,$4)`, supplier, cnpj, actor, empresa)
	testutil.Exec(t, pool, `INSERT INTO item_preferred_suppliers(enterprise_id,item_code,supplier_code,supplier_item_code,created_by) VALUES($1,$2,$3,'CH-3MM',$4),($1,$5,$3,'LUV-01',$4)`,
		empresa, chapa, supplier, actor, luva)
	var banco int64
	if err := pool.QueryRow(bg, `INSERT INTO contas_bancarias(banco,agencia,conta,descricao,saldo_inicial,is_active,created_by,enterprise_id) VALUES('001','1','1','Banco','500000',TRUE,$1,$2) RETURNING id`,
		actor, empresa).Scan(&banco); err != nil {
		t.Fatal(err)
	}

	var plano int64
	if err := pool.QueryRow(bg, `INSERT INTO accounting_plans(plan_number,description,valid_from,status) VALUES($1,'Ciclo','2020-01-01','A') RETURNING id`, u%1_000_000+500_000).Scan(&plano); err != nil {
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
	var (
		cBanco, cEstoque, cEPI, cForn                  = conta("1.1.1 BANCO"), conta("1.1.4 ESTOQUE"), conta("3.1 EPI"), conta("2.1.1 FORNECEDORES")
		cICMSr, cIPIr, cPISr, cCOFr, cIRRF             = conta("1.1.5 ICMS REC"), conta("1.1.6 IPI REC"), conta("1.1.7 PIS REC"), conta("1.1.8 COF REC"), conta("2.1.3 IRRF")
		cJurosP, cDescO, cClientes, cJurosR, cDescC    = conta("3.9 JUROS PAGOS"), conta("4.9 DESC OBTIDOS"), conta("1.1.2 CLIENTES"), conta("4.8 JUROS REC"), conta("3.8 DESC CONC")
		cReceita, cICMSv, cICMSrec, cIPIrec, cPISv     = conta("4.1 RECEITA"), conta("3.2 ICMS VENDAS"), conta("2.1.4 ICMS RECOLHER"), conta("2.1.5 IPI RECOLHER"), conta("3.3 PIS VENDAS")
		cPISrec, cCOFv, cCOFrec, cCMV, cEstoqueVendido = conta("2.1.6 PIS RECOLHER"), conta("3.4 COF VENDAS"), conta("2.1.7 COF RECOLHER"), conta("3.5 CMV"), conta("1.1.4.9 ESTOQUE PA")
	)
	var planoMP, planoEPI int64
	for _, x := range []struct {
		cod  string
		dest *int64
		cc   int64
	}{{"MP", &planoMP, cEstoque}, {"EPI", &planoEPI, cEPI}} {
		if err := pool.QueryRow(bg, `INSERT INTO plano_contas(codigo,descricao,tipo,natureza,enterprise_id,accounting_account_id) VALUES($1,$1,'A','D',$2,$3) RETURNING id`,
			fmt.Sprintf("C%d.%s", u, x.cod), empresa, x.cc).Scan(x.dest); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM accounting_journal_entries WHERE empresa_id=$1`, `DELETE FROM accounting_posting_params WHERE enterprise_id=$1`,
			`DELETE FROM notification_outbox WHERE enterprise_id=$1`, `DELETE FROM tax_assessments WHERE enterprise_id=$1`,
			`DELETE FROM fluxo_caixa WHERE enterprise_id=$1`, `DELETE FROM contas_receber WHERE enterprise_id=$1`,
			`DELETE FROM fiscal_exit_items WHERE fiscal_exit_id IN (SELECT id FROM fiscal_exits WHERE enterprise_id=$1)`,
			`DELETE FROM fiscal_exits WHERE enterprise_id=$1`, `DELETE FROM fiscal_entries WHERE enterprise_id=$1`,
			`DELETE FROM contas_pagar WHERE enterprise_id=$1`, `DELETE FROM stock_movements WHERE enterprise_id=$1`,
			`DELETE FROM stock_balances WHERE enterprise_id=$1`, `DELETE FROM contas_bancarias WHERE enterprise_id=$1`,
			`DELETE FROM item_preferred_suppliers WHERE enterprise_id=$1`, `DELETE FROM suppliers WHERE enterprise_id=$1`,
			`DELETE FROM plano_contas WHERE enterprise_id=$1`, `DELETE FROM items WHERE enterprise_id=$1`,
			`DELETE FROM warehouse WHERE enterprise_id=$1`, `DELETE FROM fiscal_configs WHERE enterprise_id=$1`,
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

	params := repository.AccountingPostingParams{
		PlanID: plano, FornecedoresAccountID: cForn, ContabilizarEntrada: true, ContabilizarPagamentos: true,
		ContabilizarRecebimentos: true, ContabilizarSaidas: true,
		ICMSRecuperarAccountID: &cICMSr, IPIRecuperarAccountID: &cIPIr, PISRecuperarAccountID: &cPISr, COFINSRecuperarAccountID: &cCOFr,
		IRRFRecolherAccountID: &cIRRF, JurosPagosAccountID: &cJurosP, DescontosObtidosAccountID: &cDescO,
		ClientesAccountID: &cClientes, JurosRecebidosAccountID: &cJurosR, DescontosConcedidosAccountID: &cDescC,
		ReceitaVendasAccountID: &cReceita, ICMSVendasAccountID: &cICMSv, ICMSRecolherAccountID: &cICMSrec, IPIRecolherAccountID: &cIPIrec,
		PISVendasAccountID: &cPISv, PISRecolherAccountID: &cPISrec, COFINSVendasAccountID: &cCOFv, COFINSRecolherAccountID: &cCOFrec,
		CMVAccountID: &cCMV, EstoqueAccountID: &cEstoqueVendido,
	}
	contab := &AccountingParamsUseCase{Docs: docs, Auth: auth}
	if _, err := contab.Save(ctx, params); err != nil {
		t.Fatalf("parâmetros: %v", err)
	}
	// Banco sem conta contábil e sem banco padrão: o pagamento é recusado com o que falta.
	baixarP := &financial_uc.BaixarContaPagarUseCase{Repo: finRepo, Auth: auth, FiscalRepo: fiscalRepo, Contabil: pg}
	if err := contab.VincularContaBancaria(ctx, banco, &cBanco); err != nil {
		t.Fatal(err)
	}

	// 1. Nota de entrada com IRRF.
	base, err := os.ReadFile("testdata/nfe_mp_epi.xml")
	if err != nil {
		t.Fatal(err)
	}
	s := strings.ReplaceAll(string(base), "41261012345678000190550010000123451000123459", chave)
	s = strings.ReplaceAll(s, "12345678000190", cnpj)
	s = strings.ReplaceAll(s, "</ICMSTot>", "</ICMSTot><retTrib><vBCIRRF>100000.00</vBCIRRF><vIRRF>1500.00</vIRRF></retTrib>")
	s = strings.ReplaceAll(s, "<vDup>33333.34</vDup>", "<vDup>31833.34</vDup>")
	upload := &UploadNFEEntryUseCase{Repo: fiscalRepo, Docs: docs, Auth: auth}
	nota, err := upload.ExecuteFile(ctx, []byte(s), request.UploadNFEDTO{})
	if err != nil {
		t.Fatal(err)
	}
	conciliar := &SaveFiscalEntryConciliationUseCase{Docs: docs, Fiscal: fiscalRepo, Auth: auth}
	if nota, err = conciliar.Execute(ctx, nota.ID, request.SaveFiscalEntryConciliationDTO{Itens: []request.FiscalEntryItemConciliationDTO{
		{ID: nota.Itens[0].ID, ItemCode: &chapa, PlanoContasID: &planoMP},
		{ID: nota.Itens[1].ID, ItemCode: &luva, PlanoContasID: &planoEPI},
	}}); err != nil || !nota.PodeAprovar {
		t.Fatalf("conciliação: %v %+v", err, nota.Pendencias)
	}
	if _, err = (&ApproveFiscalEntryUseCase{FiscalRepo: fiscalRepo, Docs: docs, FinancialRepo: finRepo, Auth: auth}).Execute(ctx, request.ApproveFiscalEntryDTO{ID: nota.ID}); err != nil {
		t.Fatalf("aprovação: %v", err)
	}

	saldo := func() map[int64]decimal.Decimal {
		rows, err := pool.Query(bg, `SELECT debit_account_id, credit_account_id, value FROM accounting_journal_entries WHERE empresa_id=$1`, empresa)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		m := map[int64]decimal.Decimal{}
		for rows.Next() {
			var d, c int64
			var v decimal.Decimal
			if err := rows.Scan(&d, &c, &v); err != nil {
				t.Fatal(err)
			}
			m[d] = m[d].Add(v)
			m[c] = m[c].Sub(v)
		}
		return m
	}
	titulo := func(where string, args ...any) (id int64, bruto decimal.Decimal) {
		if err := pool.QueryRow(bg, `SELECT id, valor_bruto FROM contas_pagar WHERE enterprise_id=$1 AND `+where+` ORDER BY id LIMIT 1`, append([]any{empresa}, args...)...).Scan(&id, &bruto); err != nil {
			t.Fatalf("título (%s): %v", where, err)
		}
		return
	}

	// 2. Duplicata 1 (33.333,33) paga em parte, com atraso e desconto.
	dup1, _ := titulo(`numero_documento LIKE 'NF-12345/1%' AND tipo_documento='NFE'`)
	testutil.Exec(t, pool, `UPDATE contas_pagar SET data_vencimento=$2::date WHERE id=$1`, dup1, time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	hoje := time.Now().Format("2006-01-02")
	if err := baixarP.Execute(ctx, dup1, request.BaixarContaPagarDTO{ContaBancariaID: banco, ValorPago: 20000, Desconto: decimal.RequireFromString("333.33"), DataPagamento: hoje}); err != nil {
		t.Fatalf("baixa parcial: %v", err)
	}
	if err := baixarP.Execute(ctx, dup1, request.BaixarContaPagarDTO{ContaBancariaID: banco, ValorPago: 1, DataPagamento: hoje}); err == nil {
		t.Fatal("pagar de novo um título já pago deveria ser recusado")
	}
	resto, brutoResto := titulo(`parcela_pai_id=$2`, dup1)
	if !brutoResto.Equal(decimal.RequireFromString("13000")) {
		t.Fatalf("saldo da baixa parcial = %s, esperado 13.000", brutoResto)
	}
	var rateioResto decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT COALESCE(SUM(valor),0) FROM contas_pagar_rateios WHERE conta_pagar_id=$1 AND enterprise_id=$2`, resto, empresa).Scan(&rateioResto); err != nil {
		t.Fatal(err)
	}
	if !rateioResto.Equal(brutoResto) {
		t.Fatalf("o saldo deveria levar o rateio por plano: rateio %s, título %s", rateioResto, brutoResto)
	}
	var juros decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT juros+multa FROM contas_pagar WHERE id=$1`, dup1).Scan(&juros); err != nil {
		t.Fatal(err)
	}
	// 30 dias de atraso: 1% de juros + 2% de multa sobre o que foi quitado agora
	// (20.000 pagos + 333,33 de desconto), não sobre o saldo inteiro.
	if want := decimal.RequireFromString("20333.33").Mul(decimal.RequireFromString("0.03")).Round(2); !juros.Equal(want) {
		t.Fatalf("juros+multa da baixa parcial = %s, esperado %s", juros, want)
	}
	if err := baixarP.Execute(ctx, resto, request.BaixarContaPagarDTO{ContaBancariaID: banco, ValorPago: 13000, DataPagamento: hoje}); err != nil {
		t.Fatalf("baixa do saldo: %v", err)
	}
	var jurosResto decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT juros+multa FROM contas_pagar WHERE id=$1`, resto).Scan(&jurosResto); err != nil {
		t.Fatal(err)
	}
	if want := decimal.RequireFromString("390"); !jurosResto.Equal(want) {
		t.Fatalf("juros+multa do saldo = %s, esperado %s (3%% de 13.000)", jurosResto, want)
	}
	juros = juros.Add(jurosResto)

	// 3. Recolhimento do IRRF.
	ret, _ := titulo(`tipo_documento='RETENCAO'`)
	if err := baixarP.Execute(ctx, ret, request.BaixarContaPagarDTO{ContaBancariaID: banco, ValorPago: 1500, DataPagamento: hoje}); err != nil {
		t.Fatalf("recolhimento do IRRF: %v", err)
	}

	m := saldo()
	// Fornecedores: 98.500 provisionados − 33.333,33 da duplicata 1 (20.000 + 333,33 de desconto + 13.000).
	if !m[cForn].Equal(decimal.RequireFromString("-65166.67")) {
		t.Fatalf("Fornecedores = %s, esperado −65.166,67", m[cForn])
	}
	if !m[cIRRF].IsZero() {
		t.Fatalf("IRRF recolhido deveria zerar: %s", m[cIRRF])
	}
	if !m[cDescO].Equal(decimal.RequireFromString("-333.33")) || !m[cJurosP].Equal(juros) {
		t.Fatalf("desconto obtido %s, juros pagos %s (esperado %s)", m[cDescO], m[cJurosP], juros)
	}
	if want := decimal.RequireFromString("-34500").Sub(juros); !m[cBanco].Equal(want) {
		t.Fatalf("Banco = %s, esperado %s (20.000 + 13.000 + 1.500 + juros)", m[cBanco], want)
	}

	// 4. Venda de 500 kg de chapa (custo médio da entrada) e recebimento.
	var custoMedio decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT avg_cost FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, empresa, chapa).Scan(&custoMedio); err != nil {
		t.Fatal(err)
	}
	var exitID int64
	if err := pool.QueryRow(bg, `INSERT INTO fiscal_exits(numero_nf,serie,data_emissao,cfop,natureza_operacao,valor_produtos,valor_frete,valor_seguro,valor_desconto,
		valor_ipi,valor_icms,valor_pis,valor_cofins,valor_total,status,is_active,created_by,emitida_contingencia,base_icms_st,valor_icms_st,enterprise_id)
		VALUES(9001,'1',$1::date,'5101','Venda',5000,0,0,0,250,600,82.50,380,5250,'AUTHORIZED',TRUE,$2,FALSE,0,0,$3) RETURNING id`, hoje, actor, empresa).Scan(&exitID); err != nil {
		t.Fatal(err)
	}
	stock := stockpg.NewStockRepositorySQLC(pool)
	ref := stockentity.ReferenceTypeNFExit
	if _, err := stock.CreateMovement(ctx, &stockentity.StockMovement{ItemCode: chapa, WarehouseID: almox, MovementType: stockentity.MovementTypeOut,
		Quantity: 500, ReferenceType: &ref, ReferenceCode: &exitID, CreatedBy: actor}); err != nil {
		t.Fatal(err)
	}
	exit := &entity.FiscalExit{ID: exitID, NumeroNF: 9001, Serie: "1", DataEmissao: time.Now(), ValorProdutos: 5000, ValorIPI: 250, ValorICMS: 600,
		ValorPIS: 82.5, ValorCOFINS: 380, ValorTotal: 5250}
	if avisos := contabilizarSaida(ctx, pg, exit, true); len(avisos) > 0 {
		t.Fatalf("contabilização da venda: %v", avisos)
	}
	if avisos := contabilizarSaida(ctx, pg, exit, true); len(avisos) > 0 {
		t.Fatalf("repetir: %v", avisos)
	}
	cmv := custoMedio.Mul(decimal.NewFromInt(500)).Round(2)
	m = saldo()
	if !m[cReceita].Equal(decimal.RequireFromString("-5000")) || !m[cIPIrec].Equal(decimal.RequireFromString("-250")) || !m[cClientes].Equal(decimal.RequireFromString("5250")) {
		t.Fatalf("venda: receita %s IPI %s clientes %s", m[cReceita], m[cIPIrec], m[cClientes])
	}
	if !m[cCMV].Sub(cmv).Abs().LessThan(decimal.RequireFromString("0.05")) || !m[cCMV].Equal(m[cEstoqueVendido].Neg()) {
		t.Fatalf("CMV %s (esperado ~%s pelo custo médio %s), estoque %s — repetir não pode lançar em dobro", m[cCMV], cmv, custoMedio, m[cEstoqueVendido])
	}
	var crID int64
	if err := pool.QueryRow(bg, `INSERT INTO contas_receber(numero_documento,fiscal_exit_id,data_lancamento,data_emissao,data_vencimento,valor_bruto,desconto,juros,multa,valor_recebido,
		parcela_numero,parcela_total,status,is_active,criado_por,enterprise_id) VALUES('NF-9001',$1,$2::date,$2::date,$2::date,5250,0,0,0,0,1,1,'PENDENTE',TRUE,$3,$4) RETURNING id`,
		exitID, hoje, actor, empresa).Scan(&crID); err != nil {
		t.Fatal(err)
	}
	baixarR := &financial_uc.BaixarContaReceberUseCase{Repo: finRepo, Auth: auth, FiscalRepo: fiscalRepo, Contabil: pg}
	if err := baixarR.Execute(ctx, crID, request.BaixarContaReceberDTO{ContaBancariaID: banco, ValorRecebido: 5200, Desconto: decimal.NewFromInt(50), DataRecebimento: hoje}); err != nil {
		t.Fatalf("recebimento: %v", err)
	}
	m = saldo()
	if !m[cClientes].IsZero() || !m[cDescC].Equal(decimal.RequireFromString("50")) {
		t.Fatalf("recebimento: clientes %s desconto concedido %s", m[cClientes], m[cDescC])
	}

	// 5. Cancelamento da venda: estoque volta pelo mesmo custo, contabilidade estorna.
	var antes decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT quantity FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, empresa, chapa).Scan(&antes); err != nil {
		t.Fatal(err)
	}
	if avisos := desfazerSaida(ctx, pg, exit, actor); len(avisos) > 0 {
		t.Fatalf("cancelamento: %v", avisos)
	}
	if avisos := desfazerSaida(ctx, pg, exit, actor); len(avisos) > 0 {
		t.Fatalf("repetir o cancelamento: %v", avisos)
	}
	var depois decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT quantity FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, empresa, chapa).Scan(&depois); err != nil {
		t.Fatal(err)
	}
	if !depois.Sub(antes).Equal(decimal.NewFromInt(500)) {
		t.Fatalf("o cancelamento deveria devolver 500 kg (uma vez só): antes %s depois %s", antes, depois)
	}
	m = saldo()
	if !m[cReceita].IsZero() || !m[cCMV].IsZero() || !m[cICMSrec].IsZero() {
		t.Fatalf("estorno da venda: receita %s CMV %s ICMS %s", m[cReceita], m[cCMV], m[cICMSrec])
	}
	total := decimal.Zero
	for _, v := range m {
		total = total.Add(v)
	}
	if !total.IsZero() {
		t.Fatalf("balancete não fecha: %s", total)
	}
	var origens int
	if err := pool.QueryRow(bg, `SELECT COUNT(DISTINCT source_type) FROM accounting_journal_entries WHERE empresa_id=$1`, empresa).Scan(&origens); err != nil {
		t.Fatal(err)
	}
	if origens != 5 { // entrada, pagamento, recebimento, saída, estorno da saída
		t.Fatalf("origens de lançamento: %d", origens)
	}
	_ = contabilizacao.OrigemSaida
}
