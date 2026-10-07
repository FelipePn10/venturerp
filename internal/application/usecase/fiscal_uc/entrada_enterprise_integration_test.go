//go:build integration

package fiscal_uc_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	financialpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	purchaseOrderRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// Nota de entrada "enterprise", ponta a ponta, numa empresa própria com
// código ≠ id (pedido de compra usa enterprise_code; nota, enterprise_id):
//
//   - a chapa já teve 4.000 kg recebidos fisicamente pelo pedido (FINS0212);
//     a nota de 10.000 kg só dá entrada nos 6.000 restantes (3-way match) e o
//     pedido fica com 10.000 faturados e recebidos;
//   - a nota traz IRRF retido: o fornecedor recebe o líquido e o IRRF vira um
//     título próprio a recolher;
//   - a contabilização sai equilibrada: o líquido devido ao fornecedor é
//     exatamente o valor a pagar;
//   - o cancelamento estorna estoque, pedido, títulos e contabilidade;
//   - duas importações simultâneas da mesma chave geram uma nota só;
//   - outra empresa não enxerga a nota.
func TestIntegration_NotaDeEntradaEnterprise(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	u := testutil.UniqueCode()
	actor := uuid.New()
	enterpriseCode := int64(1_800_000_000 + u%90_000_000)
	var empresa int64
	if _, err := pool.Exec(bg, `INSERT INTO users(id,name,email,password) VALUES($1,'Entrada',$2,'x')`, actor, actor.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Nota de entrada enterprise') RETURNING id`, enterpriseCode).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	if empresa == enterpriseCode {
		t.Fatal("o teste precisa de código ≠ id para pegar troca de convenção")
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: empresa, EnterpriseCode: enterpriseCode})

	cnpj := fmt.Sprintf("%014d", (u+7)%100_000_000_000_000)
	supplierCode := 8_000_000_000 + u%1_000_000_000
	chapa, luva := u+10, u+11
	chave := fmt.Sprintf("41%042d", u+3)
	chaveConcorrente := fmt.Sprintf("41%042d", u+4)

	testutil.Exec(t, pool, `INSERT INTO fiscal_configs(cnpj_empresa,razao_social,uf_empresa,updated_by,enterprise_id) VALUES('98765432000110','INDUSTRIA CLIENTE','PR',$1,$2)`, actor, empresa)
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,'Almox',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`,
		fmt.Sprintf("WE-%d", u), actor, empresa).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	for _, c := range []int64{chapa, luva} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name) VALUES($1,($1::bigint)::text,$2,$3,$4,'ITEM '||$1::text)`, c, almox, actor, empresa)
	}
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'ACO E SEGURANCA LTDA',$2,$3,$4)`, supplierCode, cnpj, actor, empresa)
	testutil.Exec(t, pool, `INSERT INTO item_preferred_suppliers(enterprise_id,item_code,supplier_code,supplier_item_code,created_by) VALUES($1,$2,$3,'CH-3MM',$4),($1,$5,$3,'LUV-01',$4)`,
		empresa, chapa, supplierCode, actor, luva)

	// Plano de contas financeiro ligado ao plano contábil.
	var planoContabil int64
	if err := pool.QueryRow(bg, `INSERT INTO accounting_plans(plan_number,description,valid_from,status) VALUES($1,'Plano teste','2020-01-01','A') RETURNING id`, u%1_000_000).Scan(&planoContabil); err != nil {
		t.Fatal(err)
	}
	conta := func(numero, nome string) int64 {
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO accounting_accounts(plan_id,account_number,description,nature_code,valid_from,is_analytic) VALUES($1,$2,$3,'D','2020-01-01',TRUE) RETURNING id`,
			planoContabil, numero, nome).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	cEstoque, cEPI := conta("1.1.4.01", "Estoque MP"), conta("3.1.1.01", "EPI")
	cFornecedores, cICMS, cIPI := conta("2.1.1.01", "Fornecedores"), conta("1.1.5.01", "ICMS a recuperar"), conta("1.1.5.02", "IPI a recuperar")
	cPIS, cCOFINS, cIRRF := conta("1.1.5.03", "PIS a recuperar"), conta("1.1.5.04", "COFINS a recuperar"), conta("2.1.3.01", "IRRF a recolher")
	var planoMP, planoEPI int64
	if err := pool.QueryRow(bg, `INSERT INTO plano_contas(codigo,descricao,tipo,natureza,enterprise_id) VALUES($1,'MATERIA-PRIMA','A','D',$2) RETURNING id`, fmt.Sprintf("E%d.MP", u), empresa).Scan(&planoMP); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO plano_contas(codigo,descricao,tipo,natureza,enterprise_id) VALUES($1,'EPI','A','D',$2) RETURNING id`, fmt.Sprintf("E%d.EPI", u), empresa).Scan(&planoEPI); err != nil {
		t.Fatal(err)
	}

	// Pedido de compra 77: chapa 10.000 kg a 4,80, com 4.000 kg já recebidos.
	var pedido, linhaChapa int64
	if err := pool.QueryRow(bg, `INSERT INTO purchase_orders(order_number,enterprise_code,status,supplier_code,created_by) VALUES(77,$1,'APPROVED',$2,$3) RETURNING code`,
		enterpriseCode, supplierCode, actor).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO purchase_order_items(purchase_order_code,sequence,item_code,requested_qty,received_qty,unit_price,total_price,status)
		VALUES($1,1,$2,10000,4000,4.80,48000,'PARTIAL') RETURNING code`, pedido, chapa).Scan(&linhaChapa); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM accounting_journal_entries WHERE empresa_id=$1`,
			`DELETE FROM accounting_posting_params WHERE enterprise_id=$1`,
			`DELETE FROM notification_outbox WHERE enterprise_id=$1`,
			`DELETE FROM tax_assessments WHERE enterprise_id=$1`,
			`DELETE FROM fiscal_entries WHERE enterprise_id=$1`,
			`DELETE FROM contas_pagar WHERE enterprise_id=$1`,
			`DELETE FROM stock_movements WHERE enterprise_id=$1`,
			`DELETE FROM stock_balances WHERE enterprise_id=$1`,
			`DELETE FROM purchase_order_items WHERE purchase_order_code IN (SELECT code FROM purchase_orders WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1))`,
			`DELETE FROM purchase_orders WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`,
			`DELETE FROM item_preferred_suppliers WHERE enterprise_id=$1`,
			`DELETE FROM suppliers WHERE enterprise_id=$1`,
			`DELETE FROM plano_contas WHERE enterprise_id=$1`,
			`DELETE FROM items WHERE enterprise_id=$1`,
			`DELETE FROM warehouse WHERE enterprise_id=$1`,
			`DELETE FROM fiscal_configs WHERE enterprise_id=$1`,
		} {
			if _, err := pool.Exec(bg, q, empresa); err != nil {
				t.Logf("limpeza (%s): %v", q, err)
			}
		}
		_, _ = pool.Exec(bg, `DELETE FROM accounting_accounts WHERE plan_id=$1`, planoContabil)
		_, _ = pool.Exec(bg, `DELETE FROM accounting_plans WHERE id=$1`, planoContabil)
		if _, err := pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, empresa); err != nil {
			t.Errorf("a limpeza deixou dados da empresa de teste: %v", err)
		}
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id=$1`, actor)
	})

	auth := &infraauth.AuthService{}
	fiscalRepo := fiscalpg.NewFiscalRepositoryPG(pool)
	docs := fiscalpg.NewFiscalEntryDocumentRepositoryPG(fiscalRepo)
	finRepo := financialpg.NewFinancialRepositoryPG(pool)

	// Parâmetros contábeis e vínculo plano financeiro → conta contábil.
	contab := &fiscal_uc.AccountingParamsUseCase{Docs: docs, Auth: auth}
	if _, err := contab.Save(ctx, repository.AccountingPostingParams{
		PlanID: planoContabil, FornecedoresAccountID: cFornecedores, ContabilizarEntrada: true,
		ICMSRecuperarAccountID: &cICMS, IPIRecuperarAccountID: &cIPI, PISRecuperarAccountID: &cPIS, COFINSRecuperarAccountID: &cCOFINS,
		IRRFRecolherAccountID: &cIRRF,
	}); err != nil {
		t.Fatalf("gravando parâmetros contábeis: %v", err)
	}
	if err := contab.VincularPlano(ctx, planoMP, &cEstoque); err != nil {
		t.Fatal(err)
	}
	if err := contab.VincularPlano(ctx, planoEPI, &cEPI); err != nil {
		t.Fatal(err)
	}

	base, err := os.ReadFile("testdata/nfe_mp_epi.xml")
	if err != nil {
		t.Fatal(err)
	}
	xmlDaNota := func(ch string, numero int) []byte {
		s := strings.ReplaceAll(string(base), "41261012345678000190550010000123451000123459", ch)
		s = strings.ReplaceAll(s, "12345678000190", cnpj)
		s = strings.ReplaceAll(s, "<nNF>12345</nNF>", fmt.Sprintf("<nNF>%d</nNF>", numero))
		// IRRF de 1.500 retido: as duplicatas já vêm pelo líquido (98.500).
		s = strings.ReplaceAll(s, "</ICMSTot>", "</ICMSTot><retTrib><vBCIRRF>100000.00</vBCIRRF><vIRRF>1500.00</vIRRF></retTrib>")
		s = strings.ReplaceAll(s, "<vDup>33333.34</vDup>", "<vDup>31833.34</vDup>")
		s = strings.ReplaceAll(s, "<vPag>100000.00</vPag>", "<vPag>98500.00</vPag>")
		return []byte(s)
	}

	upload := &fiscal_uc.UploadNFEEntryUseCase{Repo: fiscalRepo, Docs: docs, Auth: auth}
	nota, err := upload.ExecuteFile(ctx, xmlDaNota(chave, 880001), request.UploadNFEDTO{})
	if err != nil {
		t.Fatalf("importando: %v", err)
	}
	if nota.ValorAPagar != 98500 || nota.TotalRetencoes != 1500 {
		t.Fatalf("valor a pagar %v, retenções %v", nota.ValorAPagar, nota.TotalRetencoes)
	}
	ch := nota.Itens[0]
	if ch.ItemCode == nil || *ch.ItemCode != chapa || ch.PurchaseOrderItemCode == nil || *ch.PurchaseOrderItemCode != linhaChapa {
		t.Fatalf("a chapa deveria vir conciliada e ligada à linha do pedido 77: %+v", ch)
	}
	if ch.PurchaseOrderNumber == nil || *ch.PurchaseOrderNumber != 77 || ch.PurchaseOrderSequence == nil || *ch.PurchaseOrderSequence != 1 {
		t.Fatalf("a resposta deveria trazer o número do pedido (77) e a linha (1): %v / %v", ch.PurchaseOrderNumber, ch.PurchaseOrderSequence)
	}
	if deref(ch.CfopEntrada) != "1101" {
		t.Fatalf("fornecedor e empresa no PR: CFOP de entrada deveria ser 1101, veio %q", deref(ch.CfopEntrada))
	}
	if ch.WarehouseID == nil || *ch.WarehouseID != almox {
		t.Fatalf("almoxarifado padrão do item não aplicado: %v", ch.WarehouseID)
	}

	// Linhas de pedido em aberto para o item.
	linhas, err := (&fiscal_uc.PedidosDoItemUseCase{Docs: docs, Auth: auth}).Execute(ctx, nota.ID, ch.ID, nil)
	if err != nil || len(linhas) != 1 || linhas[0].Code != linhaChapa {
		t.Fatalf("linhas do pedido para a chapa: %+v (%v)", linhas, err)
	}

	conciliar := &fiscal_uc.SaveFiscalEntryConciliationUseCase{Docs: docs, Fiscal: fiscalRepo, Auth: auth}
	zero := int64(0)
	itensSemPlano := func(linha *int64) []request.FiscalEntryItemConciliationDTO {
		return []request.FiscalEntryItemConciliationDTO{
			{ID: nota.Itens[0].ID, ItemCode: &chapa, PurchaseOrderItemCode: linha},
			{ID: nota.Itens[1].ID, ItemCode: &luva},
		}
	}
	// Desfazer o vínculo com o pedido vale: o xPed não religa sozinho, nem nesta
	// gravação nem na próxima. Escolher a linha à mão religa.
	nota, err = conciliar.Execute(ctx, nota.ID, request.SaveFiscalEntryConciliationDTO{Itens: itensSemPlano(&zero)})
	if err != nil || nota.Itens[0].PurchaseOrderItemCode != nil {
		t.Fatalf("desvincular do pedido: linha %v (%v)", nota.Itens[0].PurchaseOrderItemCode, err)
	}
	nota, err = conciliar.Execute(ctx, nota.ID, request.SaveFiscalEntryConciliationDTO{Itens: itensSemPlano(nil)})
	if err != nil || nota.Itens[0].PurchaseOrderItemCode != nil {
		t.Fatalf("o vínculo automático refez o que o usuário desfez: linha %v (%v)", nota.Itens[0].PurchaseOrderItemCode, err)
	}
	nota, err = conciliar.Execute(ctx, nota.ID, request.SaveFiscalEntryConciliationDTO{Itens: itensSemPlano(&linhaChapa)})
	if err != nil || nota.Itens[0].PurchaseOrderItemCode == nil || *nota.Itens[0].PurchaseOrderItemCode != linhaChapa {
		t.Fatalf("religar à linha do pedido: %v (%v)", nota.Itens[0].PurchaseOrderItemCode, err)
	}

	nota, err = conciliar.Execute(ctx, nota.ID, request.SaveFiscalEntryConciliationDTO{Itens: []request.FiscalEntryItemConciliationDTO{
		{ID: nota.Itens[0].ID, ItemCode: &chapa, PlanoContasID: &planoMP},
		{ID: nota.Itens[1].ID, ItemCode: &luva, PlanoContasID: &planoEPI},
	}})
	if err != nil {
		t.Fatalf("conciliando: %v", err)
	}
	if !nota.PodeAprovar {
		t.Fatalf("nota deveria estar pronta: pendências %+v divergências %+v", nota.Pendencias, nota.Divergencias)
	}

	// Outra empresa não enxerga a nota.
	outra := testutil.TenantContext(t, pool)
	if _, err := docs.GetEntryDocument(outra, nota.ID); err == nil {
		t.Fatal("a nota de uma empresa não pode ser lida por outra")
	}

	aprovar := &fiscal_uc.ApproveFiscalEntryUseCase{FiscalRepo: fiscalRepo, Docs: docs, FinancialRepo: finRepo, Auth: auth}

	// Pedido que voltou a rascunho: a nota não dá entrada contra ele.
	if _, err := pool.Exec(bg, `UPDATE purchase_orders SET status='DRAFT' WHERE code=$1`, pedido); err != nil {
		t.Fatal(err)
	}
	conf, err := conciliar.Execute(ctx, nota.ID, request.SaveFiscalEntryConciliationDTO{Itens: []request.FiscalEntryItemConciliationDTO{
		{ID: nota.Itens[0].ID, ItemCode: &chapa, PlanoContasID: &planoMP, PurchaseOrderItemCode: &linhaChapa},
		{ID: nota.Itens[1].ID, ItemCode: &luva, PlanoContasID: &planoEPI},
	}})
	if err != nil {
		t.Fatal(err)
	}
	apontou := false
	for _, dv := range conf.Divergencias {
		apontou = apontou || (dv.Nivel == "IMPEDE" && strings.Contains(dv.Mensagem, "não foi aprovado"))
	}
	if conf.PodeAprovar || !apontou {
		t.Fatalf("pedido em rascunho deveria impedir a nota: pode=%v divergências=%+v", conf.PodeAprovar, conf.Divergencias)
	}
	if _, err := aprovar.Execute(ctx, request.ApproveFiscalEntryDTO{ID: nota.ID}); err == nil {
		t.Fatal("aprovar nota ligada a pedido em rascunho deveria falhar")
	}
	if ls, _ := (&fiscal_uc.PedidosDoItemUseCase{Docs: docs, Auth: auth}).Execute(ctx, nota.ID, ch.ID, nil); len(ls) != 0 {
		t.Fatalf("pedido em rascunho não deveria ser oferecido para ligar: %+v", ls)
	}
	if _, err := pool.Exec(bg, `UPDATE purchase_orders SET status='PARTIAL' WHERE code=$1`, pedido); err != nil {
		t.Fatal(err)
	}

	nota, err = aprovar.Execute(ctx, request.ApproveFiscalEntryDTO{ID: nota.ID})
	if err != nil {
		t.Fatalf("aprovando: %v", err)
	}

	// Consultas de compras sobre a nota aprovada: notas por linha e histórico de preço.
	consultas := purchaseOrderRepo.NewConsultasPG(pool, fiscalRepo)
	notas, err := consultas.NotasDoPedido(ctx, pedido)
	if err != nil || len(notas) != 1 || notas[0].LineCode != linhaChapa || notas[0].NumeroNF != 880001 || notas[0].Status != "APPROVED" {
		t.Fatalf("notas do pedido: %+v (%v)", notas, err)
	}
	compras, err := consultas.ComprasDoItem(ctx, chapa, time.Now().AddDate(-1, 0, 0), 10)
	if err != nil || len(compras) != 1 || compras[0].NumeroNF != 880001 || compras[0].QtdEstoque <= 0 || compras[0].CustoEstoque <= 0 {
		t.Fatalf("histórico de compras da chapa: %+v (%v)", compras, err)
	}
	if outras, _ := consultas.NotasDoPedido(outra, pedido); len(outras) != 0 {
		t.Fatalf("notas do pedido vazaram para outra empresa: %+v", outras)
	}

	saldo := func(item int64) decimal.Decimal {
		var q decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE item_code=$1 AND enterprise_id=$2`, item, empresa).Scan(&q); err != nil {
			t.Fatal(err)
		}
		return q
	}
	if !saldo(chapa).Equal(decimal.NewFromInt(6000)) {
		t.Fatalf("3-way: só os 6.000 kg ainda não recebidos entram pela nota, entrou %s", saldo(chapa))
	}
	if !saldo(luva).Equal(decimal.NewFromInt(100)) {
		t.Fatalf("luva sem pedido entra inteira: %s", saldo(luva))
	}
	var recebido, faturado decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT received_qty, invoiced_qty FROM purchase_order_items WHERE code=$1`, linhaChapa).Scan(&recebido, &faturado); err != nil {
		t.Fatal(err)
	}
	if !recebido.Equal(decimal.NewFromInt(10000)) || !faturado.Equal(decimal.NewFromInt(10000)) {
		t.Fatalf("linha do pedido: recebido %s, faturado %s (esperado 10.000 e 10.000)", recebido, faturado)
	}

	// Títulos: fornecedor pelo líquido; IRRF num título próprio.
	var fornecedor, retencao decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT COALESCE(SUM(valor_bruto) FILTER (WHERE tipo_documento<>'RETENCAO'),0), COALESCE(SUM(valor_bruto) FILTER (WHERE tipo_documento='RETENCAO'),0)
		FROM contas_pagar WHERE enterprise_id=$1 AND status<>'CANCELADO'`, empresa).Scan(&fornecedor, &retencao); err != nil {
		t.Fatal(err)
	}
	if !fornecedor.Equal(decimal.NewFromInt(98500)) || !retencao.Equal(decimal.NewFromInt(1500)) {
		t.Fatalf("títulos: fornecedor %s (esperado 98.500), retenção %s (esperado 1.500)", fornecedor, retencao)
	}

	// Contabilidade: o saldo líquido de Fornecedores é o valor a pagar.
	fornecedoresLiquido := func(tipo string) decimal.Decimal {
		var v decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT COALESCE(SUM(CASE WHEN credit_account_id=$2 THEN value ELSE 0 END) - SUM(CASE WHEN debit_account_id=$2 THEN value ELSE 0 END),0)
			FROM accounting_journal_entries WHERE empresa_id=$1 AND source_type=$3 AND source_id=$4`, empresa, cFornecedores, tipo, nota.ID).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	if v := fornecedoresLiquido("NFE_ENTRADA"); !v.Equal(decimal.NewFromInt(98500)) {
		t.Fatalf("crédito líquido em Fornecedores = %s, esperado 98.500", v)
	}
	var icmsRecuperar decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT COALESCE(SUM(value),0) FROM accounting_journal_entries WHERE empresa_id=$1 AND debit_account_id=$2`, empresa, cICMS).Scan(&icmsRecuperar); err != nil {
		t.Fatal(err)
	}
	if !icmsRecuperar.Equal(decimal.NewFromInt(5820)) {
		t.Fatalf("ICMS a recuperar = %s, esperado 5.820", icmsRecuperar)
	}

	// Cancelamento: estoque, pedido, títulos e contabilidade voltam.
	cancelar := &fiscal_uc.CancelFiscalEntryUseCase{Docs: docs, Fiscal: fiscalRepo, FinancialRepo: finRepo, Auth: auth}
	if _, err := cancelar.Execute(ctx, nota.ID, request.CancelFiscalEntryDTO{Motivo: "mercadoria devolvida integralmente"}); err != nil {
		t.Fatalf("cancelando: %v", err)
	}
	if !saldo(chapa).IsZero() || !saldo(luva).IsZero() {
		t.Fatalf("estoque após cancelar: chapa %s luva %s", saldo(chapa), saldo(luva))
	}
	if err := pool.QueryRow(bg, `SELECT received_qty, invoiced_qty FROM purchase_order_items WHERE code=$1`, linhaChapa).Scan(&recebido, &faturado); err != nil {
		t.Fatal(err)
	}
	if !recebido.Equal(decimal.NewFromInt(4000)) || !faturado.IsZero() {
		t.Fatalf("pedido após cancelar: recebido %s (esperado os 4.000 físicos), faturado %s", recebido, faturado)
	}
	var situacao string
	if err := pool.QueryRow(bg, `SELECT status FROM purchase_orders WHERE code=$1`, pedido).Scan(&situacao); err != nil {
		t.Fatal(err)
	}
	if situacao != "PARTIAL" {
		t.Fatalf("situação do pedido após o estorno = %q (os 4.000 físicos continuam recebidos)", situacao)
	}
	if v := fornecedoresLiquido("NFE_ENTRADA").Add(fornecedoresLiquido("NFE_ENTRADA_ESTORNO")); !v.IsZero() {
		t.Fatalf("contabilidade não zerou após o estorno: %s", v)
	}
	var abertos int
	if err := pool.QueryRow(bg, `SELECT COUNT(*) FROM contas_pagar WHERE enterprise_id=$1 AND status<>'CANCELADO'`, empresa).Scan(&abertos); err != nil {
		t.Fatal(err)
	}
	if abertos != 0 {
		t.Fatalf("%d título(s) continuam abertos", abertos)
	}

	// Concorrência: a mesma chave importada duas vezes ao mesmo tempo.
	var wg sync.WaitGroup
	// Vinte ao mesmo tempo, com largada única: sem a trava não bloqueante, os
	// que esperavam seguravam as conexões do pool e o dono da trava travava.
	largada := make(chan struct{})
	erros := make([]error, 20)
	for i := range erros {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-largada
			_, erros[i] = upload.ExecuteFile(ctx, xmlDaNota(chaveConcorrente, 880002), request.UploadNFEDTO{})
		}(i)
	}
	close(largada)
	wg.Wait()
	ok, conflitos := 0, 0
	for _, e := range erros {
		var c *errorsuc.ConflictError
		switch {
		case e == nil:
			ok++
		case errors.As(e, &c):
			conflitos++
		default:
			t.Fatalf("erro inesperado na importação concorrente: %v", e)
		}
	}
	if ok != 1 || conflitos != len(erros)-1 {
		t.Fatalf("importação concorrente: %d sucesso(s), %d conflito(s)", ok, conflitos)
	}
}
