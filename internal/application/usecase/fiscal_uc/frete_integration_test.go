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
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	financialpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	stockpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func TestIntegration_FreteSobreCompras(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	u := testutil.UniqueCode()
	actor := uuid.New()
	code := int64(1_300_000_000 + u%90_000_000)
	var empresa int64
	if _, err := pool.Exec(bg, `INSERT INTO users(id,name,email,password) VALUES($1,'Frete',$2,'x')`, actor, actor.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Frete sobre compras') RETURNING id`, code).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: empresa, EnterpriseCode: code})
	cnpj := "12345678000190" // o fixture do CT-e cita a NF deste emitente
	supplier := 6_000_000_000 + u%1_000_000_000
	transportadora := supplier + 1
	chapa, luva := u+40, u+41

	testutil.Exec(t, pool, `INSERT INTO fiscal_configs(cnpj_empresa,razao_social,uf_empresa,updated_by,enterprise_id) VALUES('98765432000110','INDUSTRIA','PR',$1,$2)`, actor, empresa)
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,'Almox',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`,
		fmt.Sprintf("WF-%d", u), actor, empresa).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	for _, c := range []int64{chapa, luva} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name) VALUES($1,($1::bigint)::text,$2,$3,$4,'ITEM '||$1::text)`, c, almox, actor, empresa)
	}
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'ACO LTDA',$2,$3,$4),($5,'TRANSPORTES RAPIDO',$6,$3,$4)`,
		supplier, cnpj, actor, empresa, transportadora, "11222333000181")
	testutil.Exec(t, pool, `INSERT INTO item_preferred_suppliers(enterprise_id,item_code,supplier_code,supplier_item_code,created_by) VALUES($1,$2,$3,'CH-3MM',$4),($1,$5,$3,'LUV-01',$4)`,
		empresa, chapa, supplier, actor, luva)
	var banco int64
	if err := pool.QueryRow(bg, `INSERT INTO contas_bancarias(banco,agencia,conta,descricao,saldo_inicial,is_active,created_by,enterprise_id) VALUES('001','1','1','Banco','500000',TRUE,$1,$2) RETURNING id`,
		actor, empresa).Scan(&banco); err != nil {
		t.Fatal(err)
	}
	var plano int64
	if err := pool.QueryRow(bg, `INSERT INTO accounting_plans(plan_number,description,valid_from,status) VALUES($1,'Frete','2020-01-01','A') RETURNING id`, u%1_000_000+700_000).Scan(&plano); err != nil {
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
	cBanco, cEstoque, cEPI, cForn, cICMS, cDesp := conta("BANCO"), conta("ESTOQUE"), conta("EPI"), conta("FORNECEDORES"), conta("ICMS REC"), conta("DESPESA")
	var planoMP, planoEPI int64
	if err := pool.QueryRow(bg, `INSERT INTO plano_contas(codigo,descricao,tipo,natureza,enterprise_id,accounting_account_id) VALUES($1,'MP','A','D',$2,$3) RETURNING id`,
		fmt.Sprintf("F%d.MP", u), empresa, cEstoque).Scan(&planoMP); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO plano_contas(codigo,descricao,tipo,natureza,enterprise_id,accounting_account_id) VALUES($1,'EPI','A','D',$2,$3) RETURNING id`,
		fmt.Sprintf("F%d.EPI", u), empresa, cEPI).Scan(&planoEPI); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM accounting_journal_entries WHERE empresa_id=$1`, `DELETE FROM accounting_posting_params WHERE enterprise_id=$1`,
			`DELETE FROM notification_outbox WHERE enterprise_id=$1`, `DELETE FROM tax_assessments WHERE enterprise_id=$1`,
			`DELETE FROM fluxo_caixa WHERE enterprise_id=$1`, `DELETE FROM fiscal_freight_allocations WHERE enterprise_id=$1`,
			`UPDATE fiscal_freight_documents SET conta_pagar_id=NULL WHERE enterprise_id=$1`,
			`DELETE FROM contas_pagar WHERE enterprise_id=$1`, `DELETE FROM fiscal_freight_documents WHERE enterprise_id=$1`,
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
		ContabilizarPagamentos: true, ICMSRecuperarAccountID: &cICMS, DespesaPadraoAccountID: &cDesp}); err != nil {
		t.Fatal(err)
	}
	if err := contab.VincularContaBancaria(ctx, banco, &cBanco); err != nil {
		t.Fatal(err)
	}

	// Nota de entrada aprovada (sem retenção).
	base, err := os.ReadFile("testdata/nfe_mp_epi.xml")
	if err != nil {
		t.Fatal(err)
	}
	nota, err := (&UploadNFEEntryUseCase{Repo: fiscalRepo, Docs: docs, Auth: auth}).ExecuteFile(ctx, base, request.UploadNFEDTO{})
	if err != nil {
		t.Fatal(err)
	}
	if nota, err = (&SaveFiscalEntryConciliationUseCase{Docs: docs, Fiscal: fiscalRepo, Auth: auth}).Execute(ctx, nota.ID, request.SaveFiscalEntryConciliationDTO{
		Itens: []request.FiscalEntryItemConciliationDTO{{ID: nota.Itens[0].ID, ItemCode: &chapa, PlanoContasID: &planoMP}, {ID: nota.Itens[1].ID, ItemCode: &luva, PlanoContasID: &planoEPI}},
	}); err != nil || !nota.PodeAprovar {
		t.Fatalf("conciliação: %v %+v", err, nota.Pendencias)
	}
	uc := &FreteUseCase{Repo: fiscalpg.NewFreightRepositoryPG(fiscalRepo), Docs: docs, Fiscal: fiscalRepo, FinancialRepo: finRepo, Auth: auth}
	cte, _ := os.ReadFile("testdata/cte_frete.xml")

	// CT-e antes da aprovação: liga a nota, mas o lançamento espera a aprovação.
	fr, err := uc.ImportarXML(ctx, cte, "", "")
	if err != nil {
		t.Fatalf("importando o CT-e: %v", err)
	}
	if len(fr.Notas) != 1 || fr.Notas[0].FiscalEntryID != nota.ID || fr.SupplierCode == nil || *fr.SupplierCode != transportadora {
		t.Fatalf("CT-e importado: notas %+v transportadora %v", fr.Notas, fr.SupplierCode)
	}
	if fr.CustoFrete != 1056 || fr.ValorICMS != 144 {
		t.Fatalf("custo do frete %v (esperado 1.056 = 1.200 − 144 de ICMS)", fr.CustoFrete)
	}
	if _, err := uc.Lancar(ctx, fr.ID); err == nil || !strings.Contains(err.Error(), "aprove") {
		t.Fatalf("lançar com a nota pendente deveria pedir a aprovação: %v", err)
	}
	if _, err := uc.ImportarXML(ctx, cte, "", ""); err == nil {
		t.Fatal("o mesmo CT-e duas vezes deveria ser recusado")
	} else if _, ok := err.(*errorsuc.ConflictError); !ok {
		t.Fatalf("esperava conflito: %T %v", err, err)
	}
	outroTomador := strings.Replace(string(cte), "<toma3><toma>3</toma></toma3>", "<toma3><toma>0</toma></toma3>", 1)
	outroTomador = strings.Replace(outroTomador, "4561000004560", "4571000004570", -1)
	if _, err := uc.ImportarXML(ctx, []byte(outroTomador), "", ""); err == nil || !strings.Contains(err.Error(), "tomador") {
		t.Fatalf("CT-e pago pelo remetente não é da empresa: %v", err)
	}

	if _, err = (&ApproveFiscalEntryUseCase{FiscalRepo: fiscalRepo, Docs: docs, FinancialRepo: finRepo, Auth: auth}).Execute(ctx, request.ApproveFiscalEntryDTO{ID: nota.ID}); err != nil {
		t.Fatalf("aprovação: %v", err)
	}
	// 60 das 100 caixas de luva já foram consumidas.
	ref := stockentity.ReferenceTypeManual
	if _, err := stockpg.NewStockRepositorySQLC(pool).CreateMovement(ctx, &stockentity.StockMovement{ItemCode: luva, WarehouseID: almox,
		MovementType: stockentity.MovementTypeOut, Quantity: 60, ReferenceType: &ref, CreatedBy: actor}); err != nil {
		t.Fatal(err)
	}
	custo := func(item int64) decimal.Decimal {
		var v decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT total_cost FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, empresa, item).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	antesChapa, antesLuva := custo(chapa), custo(luva)

	prev, err := uc.Obter(ctx, fr.ID)
	if err != nil || len(prev.Previa) != 2 {
		t.Fatalf("prévia do rateio: %v %+v", err, prev)
	}
	lancado, err := uc.Lancar(ctx, fr.ID)
	if err != nil {
		t.Fatalf("lançando o frete: %v", err)
	}
	if lancado.Status != repository.FreteStatusLancado || lancado.ContaPagarID == nil || len(lancado.Alocacoes) != 2 {
		t.Fatalf("frete lançado: %+v", lancado)
	}
	// A listagem traz as notas de cada frete (VFIS0210 mostra os números).
	lista, err := uc.Listar(ctx, "LANCADO")
	if err != nil || len(lista) != 1 || len(lista[0].Notas) != len(lancado.Notas) || len(lista[0].Notas) == 0 {
		t.Fatalf("listagem dos fretes sem as notas: %v %+v", err, lista)
	}
	if _, err := uc.Lancar(ctx, fr.ID); err == nil {
		t.Fatal("lançar duas vezes deveria ser recusado")
	}
	if d := custo(chapa).Sub(antesChapa); !d.Round(2).Equal(decimal.RequireFromString("528")) {
		t.Fatalf("custo da chapa subiu %s, esperado 528", d)
	}
	if d := custo(luva).Sub(antesLuva); !d.Round(2).Equal(decimal.RequireFromString("211.2")) {
		t.Fatalf("custo da luva subiu %s, esperado 211,20 (40%% de 528 ainda em estoque)", d)
	}
	var tituloValor, rateio decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT valor_bruto, (SELECT COALESCE(SUM(valor),0) FROM contas_pagar_rateios WHERE conta_pagar_id=c.id) FROM contas_pagar c WHERE id=$1`,
		*lancado.ContaPagarID).Scan(&tituloValor, &rateio); err != nil {
		t.Fatal(err)
	}
	if !tituloValor.Equal(decimal.NewFromInt(1200)) || !rateio.Equal(tituloValor) {
		t.Fatalf("título da transportadora %s, rateio %s", tituloValor, rateio)
	}
	saldo := func(conta int64, origem string) decimal.Decimal {
		var v decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT COALESCE(SUM(CASE WHEN debit_account_id=$2 THEN value ELSE 0 END) - SUM(CASE WHEN credit_account_id=$2 THEN value ELSE 0 END),0)
			FROM accounting_journal_entries WHERE empresa_id=$1 AND source_type LIKE $3`, empresa, conta, origem).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	if !saldo(cForn, "FRETE%").Equal(decimal.RequireFromString("-1200")) || !saldo(cICMS, "FRETE%").Equal(decimal.RequireFromString("144")) ||
		!saldo(cDesp, "FRETE%").Equal(decimal.RequireFromString("316.8")) {
		t.Fatalf("contabilização do frete: fornecedores %s ICMS %s despesa %s", saldo(cForn, "FRETE%"), saldo(cICMS, "FRETE%"), saldo(cDesp, "FRETE%"))
	}

	// Pagamento do título: baixa Fornecedores (o frete está provisionado).
	baixar := &financial_uc.BaixarContaPagarUseCase{Repo: finRepo, Auth: auth, FiscalRepo: fiscalRepo, Contabil: pg}
	if err := baixar.Execute(ctx, *lancado.ContaPagarID, request.BaixarContaPagarDTO{ContaBancariaID: banco, ValorPago: 1200, DataPagamento: time.Now().Format("2006-01-02")}); err != nil {
		t.Fatalf("pagando a transportadora: %v", err)
	}
	if !saldo(cForn, "PAGAMENTO_CP").Equal(decimal.RequireFromString("1200")) {
		t.Fatalf("o pagamento deveria debitar Fornecedores: %s", saldo(cForn, "PAGAMENTO_CP"))
	}
	if _, err := uc.Cancelar(ctx, fr.ID, "frete cobrado em duplicidade"); err == nil {
		t.Fatal("cancelar frete com o título pago deveria ser recusado")
	}

	// Segundo frete (manual, por quantidade), lançado e cancelado: tudo volta.
	f2, err := uc.Criar(ctx, FreteDTO{Numero: 777, Serie: "1", DataEmissao: time.Now().Format("2006-01-02"), CNPJ: "11222333000181", Nome: "TRANSPORTES RAPIDO",
		ValorFrete: decimal.NewFromInt(300), TipoRateio: "QUANTIDADE", DataVencimento: time.Now().AddDate(0, 0, 15).Format("2006-01-02"), NotasIDs: []int64{nota.ID}})
	if err != nil {
		t.Fatal(err)
	}
	antesChapa, antesLuva = custo(chapa), custo(luva)
	if _, err := uc.Lancar(ctx, f2.ID); err != nil {
		t.Fatal(err)
	}
	c2, err := uc.Cancelar(ctx, f2.ID, "transportadora errada no lançamento")
	if err != nil {
		t.Fatalf("cancelando o segundo frete: %v", err)
	}
	if c2.Status != repository.FreteStatusCancelado {
		t.Fatalf("status %s", c2.Status)
	}
	if !custo(chapa).Round(2).Equal(antesChapa.Round(2)) || !custo(luva).Round(2).Equal(antesLuva.Round(2)) {
		t.Fatalf("o cancelamento deveria devolver o custo: chapa %s→%s luva %s→%s", antesChapa, custo(chapa), antesLuva, custo(luva))
	}
	if !saldo(cForn, "FRETE_COMPRA%").Equal(decimal.RequireFromString("-1200")) {
		t.Fatalf("contabilidade do segundo frete não zerou: fornecedores %s", saldo(cForn, "FRETE_COMPRA%"))
	}
	var abertos int
	if err := pool.QueryRow(bg, `SELECT COUNT(*) FROM contas_pagar WHERE enterprise_id=$1 AND freight_document_id=$2 AND status <> 'CANCELADO'`, empresa, f2.ID).Scan(&abertos); err != nil {
		t.Fatal(err)
	}
	if abertos != 0 {
		t.Fatal("o título do frete cancelado continua aberto")
	}
	// Outra empresa não vê o frete.
	if _, err := uc.Obter(testutil.TenantContext(t, pool), fr.ID); err == nil {
		t.Fatal("outra empresa leu o frete")
	}
}
