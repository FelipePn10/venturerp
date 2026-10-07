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
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	financialpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func TestIntegration_DevolucaoDeCompra(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	u := testutil.UniqueCode()
	actor := uuid.New()
	code := int64(1_200_000_000 + u%90_000_000)
	var empresa int64
	if _, err := pool.Exec(bg, `INSERT INTO users(id,name,email,password) VALUES($1,'Devolucao',$2,'x')`, actor, actor.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Devolução de compra') RETURNING id`, code).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: empresa, EnterpriseCode: code})
	cnpj := fmt.Sprintf("%014d", (u+77)%100_000_000_000_000)
	supplier := 5_000_000_000 + u%1_000_000_000
	chapa, luva := u+60, u+61

	testutil.Exec(t, pool, `INSERT INTO fiscal_configs(cnpj_empresa,razao_social,uf_empresa,updated_by,enterprise_id) VALUES('98765432000110','INDUSTRIA','PR',$1,$2)`, actor, empresa)
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,'Almox',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`,
		fmt.Sprintf("WD-%d", u), actor, empresa).Scan(&almox); err != nil {
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
	if err := pool.QueryRow(bg, `INSERT INTO accounting_plans(plan_number,description,valid_from,status) VALUES($1,'Dev','2020-01-01','A') RETURNING id`, u%1_000_000+800_000).Scan(&plano); err != nil {
		t.Fatal(err)
	}
	conta := func(n string) int64 {
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO accounting_accounts(plan_id,account_number,description,nature_code,valid_from,is_analytic) VALUES($1,$2,$2,'D','2020-01-01',TRUE) RETURNING id`, plano, n).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	cBanco, cEstoque, cEPI, cForn, cDesp := conta("BANCO"), conta("ESTOQUE"), conta("EPI"), conta("FORN"), conta("DESPESA")
	cICMS, cIPI, cPIS, cCOF := conta("ICMS"), conta("IPI"), conta("PIS"), conta("COF")
	var planoMP, planoEPI int64
	for _, x := range []struct {
		cod  string
		dst  *int64
		acct int64
	}{{"MP", &planoMP, cEstoque}, {"EPI", &planoEPI, cEPI}} {
		if err := pool.QueryRow(bg, `INSERT INTO plano_contas(codigo,descricao,tipo,natureza,enterprise_id,accounting_account_id) VALUES($1,$1,'A','D',$2,$3) RETURNING id`,
			fmt.Sprintf("D%d.%s", u, x.cod), empresa, x.acct).Scan(x.dst); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM accounting_journal_entries WHERE empresa_id=$1`, `DELETE FROM accounting_posting_params WHERE enterprise_id=$1`,
			`DELETE FROM notification_outbox WHERE enterprise_id=$1`, `DELETE FROM tax_assessments WHERE enterprise_id=$1`,
			`DELETE FROM fluxo_caixa WHERE enterprise_id=$1`, `DELETE FROM fiscal_return_settlements WHERE enterprise_id=$1`,
			`DELETE FROM contas_receber WHERE enterprise_id=$1`,
			`DELETE FROM fiscal_exit_items WHERE fiscal_exit_id IN (SELECT id FROM fiscal_exits WHERE enterprise_id=$1)`,
			`DELETE FROM fiscal_exits WHERE enterprise_id=$1`, `DELETE FROM contas_pagar WHERE enterprise_id=$1`,
			`DELETE FROM fiscal_entries WHERE enterprise_id=$1`, `DELETE FROM stock_movements WHERE enterprise_id=$1`,
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
	contab := &AccountingParamsUseCase{Docs: docs, Auth: auth}
	if _, err := contab.Save(ctx, repository.AccountingPostingParams{PlanID: plano, FornecedoresAccountID: cForn, ContabilizarEntrada: true,
		ContabilizarPagamentos: true, ContabilizarRecebimentos: true, DespesaPadraoAccountID: &cDesp, BancoPadraoAccountID: &cBanco,
		ICMSRecuperarAccountID: &cICMS, IPIRecuperarAccountID: &cIPI, PISRecuperarAccountID: &cPIS, COFINSRecuperarAccountID: &cCOF}); err != nil {
		t.Fatal(err)
	}

	// Nota de entrada com o endereço completo do fornecedor no XML.
	base, _ := os.ReadFile("testdata/nfe_mp_epi.xml")
	// A chave contém o CNPJ: troca a chave antes do CNPJ.
	chave := fmt.Sprintf("41%042d", u+5)
	s := strings.ReplaceAll(string(base), "41261012345678000190550010000123451000123459", chave)
	s = strings.ReplaceAll(s, "12345678000190", cnpj)
	s = strings.Replace(s, "<enderEmit><xLgr>RUA A</xLgr><nro>1</nro><xMun>CURITIBA</xMun><UF>PR</UF></enderEmit>",
		"<enderEmit><xLgr>RUA A</xLgr><nro>1</nro><xBairro>CENTRO</xBairro><cMun>4106902</cMun><xMun>CURITIBA</xMun><UF>PR</UF><CEP>80000000</CEP></enderEmit>", 1)
	nota, err := (&UploadNFEEntryUseCase{Repo: fiscalRepo, Docs: docs, Auth: auth}).ExecuteFile(ctx, []byte(s), request.UploadNFEDTO{})
	if err != nil {
		t.Fatal(err)
	}
	if nota, err = (&SaveFiscalEntryConciliationUseCase{Docs: docs, Fiscal: fiscalRepo, Auth: auth}).Execute(ctx, nota.ID, request.SaveFiscalEntryConciliationDTO{
		Itens: []request.FiscalEntryItemConciliationDTO{{ID: nota.Itens[0].ID, ItemCode: &chapa, PlanoContasID: &planoMP}, {ID: nota.Itens[1].ID, ItemCode: &luva, PlanoContasID: &planoEPI}},
	}); err != nil || !nota.PodeAprovar {
		t.Fatalf("conciliação: %v %+v", err, nota.Pendencias)
	}
	dev := &DevolucaoCompraUseCase{Repo: fiscalRepo, Docs: docs, Devolucoes: fiscalpg.NewDevolucaoRepositoryPG(fiscalRepo), Estoque: pg, FinancialRepo: finRepo, Auth: auth}
	hoje := time.Now().Format("2006-01-02")
	if _, err := dev.Criar(ctx, DevolucaoCompraDTO{FiscalEntryID: nota.ID, DataEmissao: hoje, Itens: []DevolucaoItemDTO{{FiscalEntryItemID: nota.Itens[0].ID, Quantidade: decimal.NewFromInt(1)}}}); err == nil {
		t.Fatal("devolver nota não aprovada deveria ser recusado")
	}
	if _, err = (&ApproveFiscalEntryUseCase{FiscalRepo: fiscalRepo, Docs: docs, FinancialRepo: finRepo, Auth: auth}).Execute(ctx, request.ApproveFiscalEntryDTO{ID: nota.ID}); err != nil {
		t.Fatalf("aprovação: %v", err)
	}
	chapaItem, luvaItem := nota.Itens[0].ID, nota.Itens[1].ID

	previa, err := dev.Previa(ctx, nota.ID)
	if err != nil || len(previa.Itens) != 2 || previa.Itens[0].Disponivel != 10000 || previa.Itens[0].CFOPDevolucao != "5201" {
		t.Fatalf("prévia: %v %+v", err, previa)
	}
	d1, err := dev.Criar(ctx, DevolucaoCompraDTO{FiscalEntryID: nota.ID, DataEmissao: hoje,
		Itens: []DevolucaoItemDTO{{FiscalEntryItemID: chapaItem, Quantidade: decimal.NewFromInt(1000)}}})
	if err != nil {
		t.Fatalf("criando a devolução: %v", err)
	}
	if d1.Finalidade != 4 || d1.NFeReferenciada == nil || *d1.NFeReferenciada != chave || d1.Cfop != "5201" || d1.ValorTotal != 5000 || d1.ValorICMS != 582 || d1.ValorIPI != 150 {
		t.Fatalf("NF-e de devolução: finalidade %d ref %q (esperado %s) CFOP %s total %v ICMS %v IPI %v", d1.Finalidade, deref(d1.NFeReferenciada), chave, d1.Cfop, d1.ValorTotal, d1.ValorICMS, d1.ValorIPI)
	}
	exit1, err := fiscalRepo.GetExitByID(ctx, d1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if exit1.DestBairro == nil || *exit1.DestBairro != "CENTRO" || exit1.DestCEP == nil || *exit1.DestCEP != "80000000" {
		t.Fatalf("endereço do fornecedor vindo do XML: bairro %v CEP %v", exit1.DestBairro, exit1.DestCEP)
	}
	if _, err := dev.Criar(ctx, DevolucaoCompraDTO{FiscalEntryID: nota.ID, DataEmissao: hoje,
		Itens: []DevolucaoItemDTO{{FiscalEntryItemID: chapaItem, Quantidade: decimal.NewFromInt(9500)}}}); err == nil {
		t.Fatal("devolver mais do que resta (9.000) deveria ser recusado")
	}
	// Payload para a SEFAZ: finalidade 4, nota referenciada, sem pagamento.
	itens1, _ := fiscalRepo.GetExitItems(ctx, d1.ID)
	cfg, _ := fiscalRepo.GetFiscalConfig(ctx)
	payload := montarPayloadNFe(exit1, itens1, cfg, PlanoDaNota{})
	if payload.FinalidadeEmissao != 4 || len(payload.NotasReferenciadas) != 1 || payload.NotasReferenciadas[0].ChaveNFe != chave ||
		len(payload.FormaPagamento) != 1 || payload.FormaPagamento[0].FormaPagamento != "90" || payload.Duplicatas != nil {
		t.Fatalf("payload da devolução: %+v %+v %+v", payload.FinalidadeEmissao, payload.NotasReferenciadas, payload.FormaPagamento)
	}

	saldoEstoque := func(item int64) decimal.Decimal {
		var q decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, empresa, item).Scan(&q); err != nil {
			t.Fatal(err)
		}
		return q
	}
	contabil := func(conta int64, origem string) decimal.Decimal {
		var v decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT COALESCE(SUM(CASE WHEN debit_account_id=$2 THEN value ELSE 0 END) - SUM(CASE WHEN credit_account_id=$2 THEN value ELSE 0 END),0)
			FROM accounting_journal_entries WHERE empresa_id=$1 AND source_type LIKE $3`, empresa, conta, origem).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	// Autorizada na SEFAZ (simulado): efetiva.
	testutil.Exec(t, pool, `UPDATE fiscal_exits SET status='AUTHORIZED' WHERE id=$1`, d1.ID)
	exit1.Status = "AUTHORIZED"
	avisos := dev.Efetivar(ctx, exit1, actor)
	if len(avisos) != 1 || !strings.Contains(avisos[0], "5000.00 abatidos") {
		t.Fatalf("efetivação: %v", avisos)
	}
	if len(dev.Efetivar(ctx, exit1, actor)) > 1 || !saldoEstoque(chapa).Equal(decimal.NewFromInt(9000)) {
		t.Fatalf("efetivar de novo não pode repetir: estoque da chapa %s", saldoEstoque(chapa))
	}
	var descontoDup3 decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT desconto FROM contas_pagar WHERE enterprise_id=$1 AND numero_documento = 'NF-12345/1 3/3'`, empresa).Scan(&descontoDup3); err != nil {
		t.Fatal(err)
	}
	if !descontoDup3.Equal(decimal.NewFromInt(5000)) {
		t.Fatalf("a última duplicata deveria ser abatida em 5.000: %s", descontoDup3)
	}
	if !contabil(cForn, "NFE_DEVOLUCAO").Equal(decimal.NewFromInt(5000)) {
		t.Fatalf("Fornecedores na devolução: %s", contabil(cForn, "NFE_DEVOLUCAO"))
	}
	total := decimal.Zero
	for _, c := range []int64{cForn, cEstoque, cICMS, cIPI, cPIS, cCOF, cDesp} {
		total = total.Add(contabil(c, "NFE_DEVOLUCAO"))
	}
	if !total.IsZero() {
		t.Fatalf("a contabilização da devolução não fecha: %s", total)
	}

	// Títulos todos pagos; devolução de 10 caixas de luva vira crédito a receber.
	rows, err := pool.Query(bg, `SELECT id, valor_bruto - desconto FROM contas_pagar WHERE enterprise_id=$1 AND status='PENDENTE' ORDER BY id`, empresa)
	if err != nil {
		t.Fatal(err)
	}
	type ab struct {
		id    int64
		saldo decimal.Decimal
	}
	var abertos []ab
	for rows.Next() {
		var a ab
		_ = rows.Scan(&a.id, &a.saldo)
		abertos = append(abertos, a)
	}
	rows.Close()
	baixar := &financial_uc.BaixarContaPagarUseCase{Repo: finRepo, Auth: auth, FiscalRepo: fiscalRepo, Contabil: pg}
	for _, a := range abertos {
		if err := baixar.Execute(ctx, a.id, request.BaixarContaPagarDTO{ContaBancariaID: banco, ValorPago: a.saldo.InexactFloat64(), DataPagamento: hoje}); err != nil {
			t.Fatalf("pagando %d: %v", a.id, err)
		}
	}
	d2, err := dev.Criar(ctx, DevolucaoCompraDTO{FiscalEntryID: nota.ID, DataEmissao: hoje, Itens: []DevolucaoItemDTO{{FiscalEntryItemID: luvaItem, Quantidade: decimal.NewFromInt(10)}}})
	if err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `UPDATE fiscal_exits SET status='AUTHORIZED' WHERE id=$1`, d2.ID)
	exit2, _ := fiscalRepo.GetExitByID(ctx, d2.ID)
	if av := dev.Efetivar(ctx, exit2, actor); len(av) != 1 || !strings.Contains(av[0], "crédito a receber") {
		t.Fatalf("devolução com títulos pagos: %v", av)
	}
	var crID int64
	var crForn *int64
	if err := pool.QueryRow(bg, `SELECT id, fornecedor_id FROM contas_receber WHERE enterprise_id=$1 AND fiscal_exit_id=$2`, empresa, d2.ID).Scan(&crID, &crForn); err != nil {
		t.Fatal(err)
	}
	if crForn == nil || *crForn != supplier {
		t.Fatalf("o crédito deveria ser do fornecedor: %v", crForn)
	}
	if err := dev.PodeDesfazer(ctx, exit2); err != nil {
		t.Fatalf("antes do pagamento do crédito, a devolução pode ser desfeita: %v", err)
	}
	// A carteira a receber devolve o fornecedor do crédito (VFIN0210 mostra quem deve).
	if lido, err := finRepo.GetContaReceber(ctx, crID); err != nil || lido.FornecedorID == nil || *lido.FornecedorID != supplier {
		t.Fatalf("GetContaReceber sem o fornecedor: %+v %v", lido, err)
	}
	receber := &financial_uc.BaixarContaReceberUseCase{Repo: finRepo, Auth: auth, FiscalRepo: fiscalRepo, Contabil: pg}
	if err := receber.Execute(ctx, crID, request.BaixarContaReceberDTO{ContaBancariaID: banco, ValorRecebido: 5000, DataRecebimento: hoje}); err != nil {
		t.Fatalf("recebendo o crédito: %v", err)
	}
	if !contabil(cForn, "RECEBIMENTO_CR").Equal(decimal.NewFromInt(-5000)) {
		t.Fatalf("o recebimento do crédito deveria creditar Fornecedores: %s", contabil(cForn, "RECEBIMENTO_CR"))
	}
	if err := dev.PodeDesfazer(ctx, exit2); err == nil {
		t.Fatal("com o crédito recebido, a devolução não pode ser cancelada")
	}

	// Cancelamento da primeira devolução: título, estoque e contabilidade voltam.
	estoqueAntes := saldoEstoque(chapa)
	if av := dev.Desfazer(ctx, exit1, actor); len(av) > 0 {
		t.Fatalf("desfazendo: %v", av)
	}
	if !saldoEstoque(chapa).Sub(estoqueAntes).Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("o estoque da chapa deveria voltar 1.000 kg: %s → %s", estoqueAntes, saldoEstoque(chapa))
	}
	// Sobra só a devolução 2 (5.000 em Fornecedores); a 1 foi estornada.
	if v := contabil(cForn, "NFE_DEVOL%"); !v.Equal(decimal.NewFromInt(5000)) {
		t.Fatalf("estorno contábil: Fornecedores nas devoluções = %s, esperado 5.000 (só a segunda)", v)
	}
	var desc decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT desconto FROM contas_pagar WHERE enterprise_id=$1 AND numero_documento = 'NF-12345/1 3/3' AND parcela_pai_id IS NULL`, empresa).Scan(&desc); err != nil {
		t.Fatal(err)
	}
	if !desc.IsZero() {
		t.Fatalf("o abatimento deveria ser desfeito: desconto %s", desc)
	}
}
