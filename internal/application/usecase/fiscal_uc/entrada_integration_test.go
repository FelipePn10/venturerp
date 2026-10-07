//go:build integration

package fiscal_uc_test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	financialrepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	financialpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O caso do pedido, ponta a ponta: nota de 100 mil de um fornecedor com 50 mil
// de matéria-prima e 50 mil de EPI, em 3 duplicatas. O XML é importado pelo
// arquivo, os itens são conciliados com o cadastro (um pelo vínculo já
// existente, outro à mão e memorizado), cada item recebe o seu plano de
// contas, as parcelas são distribuídas por plano — inclusive de forma não
// proporcional — e a aprovação gera um título por duplicata com o rateio.
func TestIntegration_NotaDeEntradaXMLConciliacaoRateio(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := testutil.TenantContext(t, pool)
	actor := testutil.Actor(t, pool)
	empresa := testutil.EnterpriseID(t, ctx)
	auth := &infraauth.AuthService{}

	fiscalRepo := fiscalpg.NewFiscalRepositoryPG(pool)
	docs := fiscalpg.NewFiscalEntryDocumentRepositoryPG(fiscalRepo)
	finRepo := financialpg.NewFinancialRepositoryPG(pool)

	u := testutil.UniqueCode()
	cnpj := fmt.Sprintf("%014d", u%100_000_000_000_000)
	chave1 := fmt.Sprintf("41%042d", u)
	chave2 := fmt.Sprintf("42%042d", u)
	supplierCode := 9_000_000_000 + u%1_000_000_000
	chapa, luva := u, u+1

	base, err := os.ReadFile("testdata/nfe_mp_epi.xml")
	if err != nil {
		t.Fatal(err)
	}
	xmlDaNota := func(chave string, numero int) []byte {
		s := strings.ReplaceAll(string(base), "41261012345678000190550010000123451000123459", chave)
		s = strings.ReplaceAll(s, "12345678000190", cnpj)
		s = strings.ReplaceAll(s, "<nNF>12345</nNF>", fmt.Sprintf("<nNF>%d</nNF>", numero))
		return []byte(s)
	}

	// Este teste cobre a nota sem contabilização automática (a contabilização
	// tem o seu, numa empresa própria): garante que a empresa de teste não
	// ficou configurada por outra execução.
	testutil.Exec(t, pool, `DELETE FROM accounting_posting_params WHERE enterprise_id=$1`, empresa)
	testutil.SeedItem(t, pool, ctx, chapa, actor)
	testutil.SeedItem(t, pool, ctx, luva, actor)
	testutil.Exec(t, pool, `UPDATE items SET name='CHAPA DE ACO 1020 ESPESSURA 3MM' WHERE code=$1`, chapa)
	testutil.Exec(t, pool, `UPDATE items SET name='LUVA DE SEGURANCA EM RASPA' WHERE code=$1`, luva)
	// O almoxarifado padrão do item é o que a nota assume para movimentar o estoque.
	var almox int64
	if err := pool.QueryRow(ctx, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,'Almox NF',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`,
		fmt.Sprintf("WNF-%d", u), actor, empresa).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `UPDATE items SET warehouse_code=$3 WHERE code IN ($1,$2)`, chapa, luva, almox)
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'ACO E SEGURANCA LTDA',$2,$3,$4)`,
		supplierCode, cnpj, actor, empresa)
	testutil.Exec(t, pool, `INSERT INTO item_preferred_suppliers(enterprise_id,item_code,supplier_code,supplier_item_code,created_by) VALUES($1,$2,$3,'LUV-01',$4)`,
		empresa, luva, supplierCode, actor)
	var planoMP, planoEPI int64
	if err := pool.QueryRow(ctx, `INSERT INTO plano_contas(codigo,descricao,tipo,natureza,enterprise_id) VALUES($1,'MATERIA-PRIMA','A','D',$2) RETURNING id`,
		fmt.Sprintf("T%d.MP", u), empresa).Scan(&planoMP); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO plano_contas(codigo,descricao,tipo,natureza,enterprise_id) VALUES($1,'EPI','A','D',$2) RETURNING id`,
		fmt.Sprintf("T%d.EPI", u), empresa).Scan(&planoEPI); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM notification_outbox WHERE aggregate_type='NFE_ENTRADA' AND aggregate_internal_id IN (SELECT id::text FROM fiscal_entries WHERE cnpj_emitente=$1)`, cnpj)
		testutil.Exec(t, pool, `DELETE FROM fiscal_entries WHERE cnpj_emitente=$1`, cnpj)
		testutil.Exec(t, pool, `DELETE FROM contas_pagar WHERE fornecedor_cnpj=$1`, cnpj)
		testutil.Exec(t, pool, `DELETE FROM item_preferred_suppliers WHERE supplier_code=$1`, supplierCode)
		testutil.Exec(t, pool, `DELETE FROM suppliers WHERE code=$1`, supplierCode)
		testutil.Exec(t, pool, `DELETE FROM plano_contas WHERE id IN ($1,$2)`, planoMP, planoEPI)
		testutil.Exec(t, pool, `DELETE FROM stock_movements WHERE item_code IN ($1,$2)`, chapa, luva)
		testutil.Exec(t, pool, `DELETE FROM stock_balances WHERE item_code IN ($1,$2)`, chapa, luva)
		testutil.Exec(t, pool, `DELETE FROM items WHERE code IN ($1,$2)`, chapa, luva)
		testutil.Exec(t, pool, `DELETE FROM warehouse WHERE id=$1`, almox)
	})

	upload := &fiscal_uc.UploadNFEEntryUseCase{Repo: fiscalRepo, Docs: docs, Auth: auth}

	// 1. Importação do arquivo.
	nota, err := upload.ExecuteFile(ctx, xmlDaNota(chave1, 777001), request.UploadNFEDTO{})
	if err != nil {
		t.Fatalf("importando XML: %v", err)
	}
	if nota.SupplierCode == nil || *nota.SupplierCode != supplierCode {
		t.Fatalf("fornecedor não foi encontrado pelo CNPJ: %v", nota.SupplierCode)
	}
	if len(nota.Itens) != 2 || len(nota.Parcelas) != 3 {
		t.Fatalf("itens=%d parcelas=%d", len(nota.Itens), len(nota.Parcelas))
	}
	if nota.Itens[0].ItemCode != nil {
		t.Fatal("a chapa não tem vínculo e não pode ser conciliada sozinha")
	}
	if nota.Itens[1].ItemCode == nil || *nota.Itens[1].ItemCode != luva || deref(nota.Itens[1].ResolutionStrategy) != "CODIGO_EXATO" {
		t.Fatalf("a luva deveria vir conciliada pelo vínculo: %+v", nota.Itens[1])
	}
	if nota.Itens[0].ValorICMS != 5820 || nota.Itens[0].ValorContabil != 50000 {
		t.Fatalf("impostos/valor contábil da chapa: ICMS %v contábil %v", nota.Itens[0].ValorICMS, nota.Itens[0].ValorContabil)
	}
	if nota.PodeAprovar || nota.Status != "PENDING" {
		t.Fatal("nota sem conciliação/plano não pode estar liberada para aprovação")
	}

	// 2. O mesmo XML de novo é recusado.
	if _, err := upload.ExecuteFile(ctx, xmlDaNota(chave1, 777001), request.UploadNFEDTO{}); err == nil {
		t.Fatal("importar a mesma chave duas vezes deveria ser recusado")
	} else {
		var conflito *errorsuc.ConflictError
		if !errors.As(err, &conflito) {
			t.Fatalf("esperava conflito, veio %T: %v", err, err)
		}
	}

	// 3. Sugestão para a chapa encontra o item pela descrição.
	sug, err := (&fiscal_uc.SuggestFiscalEntryItemUseCase{Docs: docs, Auth: auth}).Execute(ctx, nota.ID, nota.Itens[0].ID, "")
	if err != nil {
		t.Fatal(err)
	}
	achou := false
	for _, s := range sug {
		if s.ItemCode == chapa {
			achou = true
		}
	}
	if !achou {
		t.Fatalf("a chapa do cadastro deveria ser sugerida: %+v", sug)
	}

	// 4. Conciliação e classificação.
	conciliar := &fiscal_uc.SaveFiscalEntryConciliationUseCase{Docs: docs, Auth: auth}
	nota, err = conciliar.Execute(ctx, nota.ID, request.SaveFiscalEntryConciliationDTO{Itens: []request.FiscalEntryItemConciliationDTO{
		{ID: nota.Itens[0].ID, ItemCode: &chapa, PlanoContasID: &planoMP, LembrarVinculo: true},
		{ID: nota.Itens[1].ID, ItemCode: &luva, PlanoContasID: &planoEPI},
	}})
	if err != nil {
		t.Fatalf("conciliando: %v", err)
	}
	if !nota.PodeAprovar || nota.Status != "CONFERRED" {
		t.Fatalf("nota conciliada e classificada deveria estar conferida: %+v", nota.Pendencias)
	}
	for _, p := range nota.Parcelas {
		if len(p.Distribuicao) != 2 {
			t.Fatalf("parcela %d deveria estar distribuída entre MP e EPI: %+v", p.Numero, p.Distribuicao)
		}
	}

	// 5. Distribuição não proporcional: 1ª parcela toda EPI, 2ª toda MP, 3ª o resto.
	parcelas := []request.FiscalEntryInstallmentDTO{}
	for _, p := range nota.Parcelas {
		dto := request.FiscalEntryInstallmentDTO{Numero: p.Numero, Documento: p.Documento, DataVencimento: p.DataVencimento, Valor: decimal.NewFromFloat(p.Valor), FormaPagamento: p.FormaPagamento}
		switch p.Numero {
		case 1:
			dto.Distribuicao = []request.FiscalEntryAllocationDTO{{PlanoContasID: planoEPI, Valor: decimal.RequireFromString("33333.33")}}
		case 2:
			dto.Distribuicao = []request.FiscalEntryAllocationDTO{{PlanoContasID: planoMP, Valor: decimal.RequireFromString("33333.33")}}
		case 3:
			dto.Distribuicao = []request.FiscalEntryAllocationDTO{
				{PlanoContasID: planoMP, Valor: decimal.RequireFromString("16666.67")},
				{PlanoContasID: planoEPI, Valor: decimal.RequireFromString("16666.67")},
			}
		}
		parcelas = append(parcelas, dto)
	}
	itensMantidos := []request.FiscalEntryItemConciliationDTO{
		{ID: nota.Itens[0].ID, ItemCode: &chapa, PlanoContasID: &planoMP},
		{ID: nota.Itens[1].ID, ItemCode: &luva, PlanoContasID: &planoEPI},
	}
	nota, err = conciliar.Execute(ctx, nota.ID, request.SaveFiscalEntryConciliationDTO{Itens: itensMantidos, Parcelas: parcelas})
	if err != nil {
		t.Fatalf("gravando distribuição manual: %v", err)
	}
	if !nota.PodeAprovar {
		t.Fatalf("distribuição manual válida não liberou a aprovação: %+v", nota.Pendencias)
	}

	// 6. Aprovação: um título por duplicata, com o rateio.
	aprovar := &fiscal_uc.ApproveFiscalEntryUseCase{FiscalRepo: fiscalRepo, Docs: docs, Auth: auth}
	nota, err = aprovar.Execute(ctx, request.ApproveFiscalEntryDTO{ID: nota.ID})
	if err != nil {
		t.Fatalf("aprovando: %v", err)
	}
	if nota.Status != "APPROVED" || nota.StockStatus != "CONCLUIDO" {
		t.Fatalf("status = %s, estoque = %s", nota.Status, nota.StockStatus)
	}
	saldo := func(item int64) decimal.Decimal {
		var q decimal.Decimal
		if err := pool.QueryRow(ctx, `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE item_code=$1 AND warehouse_id=$2 AND enterprise_id=$3`, item, almox, empresa).Scan(&q); err != nil {
			t.Fatal(err)
		}
		return q
	}
	if !saldo(chapa).Equal(decimal.NewFromInt(10000)) || !saldo(luva).Equal(decimal.NewFromInt(100)) {
		t.Fatalf("a aprovação deveria dar entrada no estoque: chapa %s, luva %s", saldo(chapa), saldo(luva))
	}
	for _, it := range nota.Itens {
		if it.StockMovementID == nil || it.CustoAquisicao <= 0 {
			t.Fatalf("item %d sem movimento de estoque ou custo de aquisição: %+v", it.Sequence, it)
		}
	}
	if _, err := aprovar.Execute(ctx, request.ApproveFiscalEntryDTO{ID: nota.ID}); err == nil {
		t.Fatal("aprovar duas vezes duplicaria os títulos e deveria ser recusado")
	}
	titulos, err := finRepo.ListContasPagar(ctx, financialrepo.CPFilter{Documento: strPtr("NF-777001/")})
	if err != nil {
		t.Fatal(err)
	}
	if len(titulos) != 3 {
		t.Fatalf("esperava 3 títulos, vieram %d", len(titulos))
	}
	for _, ti := range titulos {
		if ti.FornecedorID == nil || *ti.FornecedorID != supplierCode {
			t.Fatalf("título sem o fornecedor: %+v", ti)
		}
		soma := decimal.Zero
		for _, r := range ti.Rateios {
			soma = soma.Add(r.Valor)
		}
		if !soma.Equal(ti.ValorBruto) {
			t.Fatalf("rateio do título %s soma %s e o título vale %s", ti.NumeroDocumento, soma, ti.ValorBruto)
		}
		if ti.ParcelaNumero == 1 && (ti.PlanoContasID == nil || *ti.PlanoContasID != planoEPI) {
			t.Fatalf("a parcela 1 vai inteira para EPI e deveria levar o plano na capa: %+v", ti.PlanoContasID)
		}
	}
	epi, err := finRepo.ListContasPagar(ctx, financialrepo.CPFilter{Documento: strPtr("NF-777001/"), PlanoContasID: &planoEPI})
	if err != nil {
		t.Fatal(err)
	}
	if len(epi) != 2 {
		t.Fatalf("o filtro por EPI deveria achar as parcelas 1 e 3 (pelo rateio), achou %d", len(epi))
	}
	porPlano, err := finRepo.ContasPagarPorPlano(ctx, financialrepo.CPPorPlanoFilter{FornecedorID: &supplierCode})
	if err != nil {
		t.Fatal(err)
	}
	totais := map[int64]decimal.Decimal{}
	for _, l := range porPlano {
		if l.PlanoContasID != nil {
			totais[*l.PlanoContasID] = l.ValorTotal
		}
	}
	if !totais[planoMP].Equal(decimal.NewFromInt(50000)) || !totais[planoEPI].Equal(decimal.NewFromInt(50000)) {
		t.Fatalf("contas a pagar por plano: MP %s, EPI %s (esperado 50 mil cada)", totais[planoMP], totais[planoEPI])
	}

	// 7. A próxima nota do mesmo fornecedor já vem conciliada e classificada:
	// a chapa pelo vínculo memorizado e os planos pelo último uso de cada item.
	nota2, err := upload.ExecuteFile(ctx, xmlDaNota(chave2, 777002), request.UploadNFEDTO{})
	if err != nil {
		t.Fatalf("importando a segunda nota: %v", err)
	}
	if nota2.Itens[0].ItemCode == nil || *nota2.Itens[0].ItemCode != chapa {
		t.Fatalf("o vínculo memorizado não conciliou a chapa: %+v", nota2.Itens[0])
	}
	if nota2.Itens[0].PlanoContasID == nil || *nota2.Itens[0].PlanoContasID != planoMP {
		t.Fatalf("o plano de contas da chapa deveria vir do último uso: %v", nota2.Itens[0].PlanoContasID)
	}
	if !nota2.PodeAprovar || nota2.Status != "CONFERRED" {
		t.Fatalf("a segunda nota deveria nascer pronta para aprovar: %+v", nota2.Pendencias)
	}
	if nota2.Itens[0].WarehouseID == nil || *nota2.Itens[0].WarehouseID != almox {
		t.Fatalf("o almoxarifado deveria vir do cadastro do item: %v", nota2.Itens[0].WarehouseID)
	}

	// 8. Cancelamento da nota aprovada: títulos cancelados e estoque estornado.
	cancelar := &fiscal_uc.CancelFiscalEntryUseCase{Docs: docs, Fiscal: fiscalRepo, FinancialRepo: finRepo, Auth: auth}
	if _, err := cancelar.Execute(ctx, nota.ID, request.CancelFiscalEntryDTO{Motivo: "curto"}); err == nil {
		t.Fatal("cancelamento sem motivo suficiente deveria ser recusado")
	}
	cancelada, err := cancelar.Execute(ctx, nota.ID, request.CancelFiscalEntryDTO{Motivo: "nota lançada em duplicidade pelo fornecedor"})
	if err != nil {
		t.Fatalf("cancelando: %v", err)
	}
	if cancelada.Status != "CANCELLED" || cancelada.CancelReason == nil || cancelada.StockStatus != "ESTORNADO" {
		t.Fatalf("após cancelar: status %s, estoque %s", cancelada.Status, cancelada.StockStatus)
	}
	if !saldo(chapa).IsZero() || !saldo(luva).IsZero() {
		t.Fatalf("o cancelamento deveria estornar o estoque: chapa %s, luva %s", saldo(chapa), saldo(luva))
	}
	var abertos int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM contas_pagar WHERE fornecedor_cnpj=$1 AND numero_documento LIKE 'NF-777001/%' AND status <> 'CANCELADO'`, cnpj).Scan(&abertos); err != nil {
		t.Fatal(err)
	}
	if abertos != 0 {
		t.Fatalf("%d título(s) da nota cancelada continuam em aberto", abertos)
	}
	if _, err := cancelar.Execute(ctx, nota.ID, request.CancelFiscalEntryDTO{Motivo: "nota lançada em duplicidade pelo fornecedor"}); err == nil {
		t.Fatal("cancelar duas vezes deveria ser recusado")
	}
}

func strPtr(s string) *string { return &s }

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
