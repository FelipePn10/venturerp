//go:build integration

package fiscal_uc

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/supplier_uc"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	supplierrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/supplier"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// Nota de um emitente que não está no cadastro, com itens que também não
// estão: a aprovação fica travada até o fornecedor existir; o fornecedor e os
// itens se cadastram a partir da própria nota.
func TestIntegration_NotaDeFornecedorEItensSemCadastro(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	u := testutil.UniqueCode()
	actor := uuid.New()
	code := int64(1_400_000_000 + u%90_000_000)
	var empresa int64
	testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password) VALUES($1,'Cadastros da nota',$2,'x')`, actor, actor.String()+"@example.test")
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Cadastros da nota') RETURNING id`, code).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: empresa, EnterpriseCode: code})
	cnpj := cnpjComDV(fmt.Sprintf("%012d", (u+91)%1_000_000_000_000))
	testutil.Exec(t, pool, `INSERT INTO fiscal_configs(cnpj_empresa,razao_social,uf_empresa,updated_by,enterprise_id) VALUES('98765432000110','INDUSTRIA','PR',$1,$2)`, actor, empresa)
	var almox, plano int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,'Almox',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`,
		fmt.Sprintf("WC-%d", u), actor, empresa).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO plano_contas(codigo,descricao,tipo,natureza,enterprise_id) VALUES($1,'MP','A','D',$2) RETURNING id`, fmt.Sprintf("C%d", u), empresa).Scan(&plano); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM notification_outbox WHERE enterprise_id=$1`, `DELETE FROM contas_pagar WHERE enterprise_id=$1`,
			`DELETE FROM fiscal_entries WHERE enterprise_id=$1`, `DELETE FROM stock_movements WHERE enterprise_id=$1`,
			`DELETE FROM stock_balances WHERE enterprise_id=$1`, `DELETE FROM item_preferred_suppliers WHERE enterprise_id=$1`,
			`DELETE FROM supplier_addresses WHERE supplier_id IN (SELECT id FROM suppliers WHERE enterprise_id=$1)`,
			`DELETE FROM suppliers WHERE enterprise_id=$1`, `DELETE FROM items WHERE enterprise_id=$1`,
			`DELETE FROM plano_contas WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`,
			`DELETE FROM fiscal_configs WHERE enterprise_id=$1`, `DELETE FROM enterprise WHERE id=$1`,
		} {
			if _, err := pool.Exec(bg, q, empresa); err != nil {
				t.Errorf("limpeza (%s): %v", q, err)
			}
		}
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id=$1`, actor)
	})

	auth := &infraauth.AuthService{}
	fiscalRepo := fiscalpg.NewFiscalRepositoryPG(pool)
	docs := fiscalpg.NewFiscalEntryDocumentRepositoryPG(fiscalRepo)
	q := sqlc.New(pool)
	supplierUC := supplier_uc.NewSupplierUseCase(supplierrepo.New(q, pool), auth)
	itemUC := item_uc.NewCreateItemUseCase(itemrepo.NewRepositoryItemSQLC(q), auth)

	base, _ := os.ReadFile("testdata/nfe_mp_epi.xml")
	chave := fmt.Sprintf("41%042d", u+11)
	s := strings.ReplaceAll(string(base), "41261012345678000190550010000123451000123459", chave)
	s = strings.ReplaceAll(s, "12345678000190", cnpj)
	s = strings.Replace(s, "<enderEmit><xLgr>RUA A</xLgr><nro>1</nro><xMun>CURITIBA</xMun><UF>PR</UF></enderEmit>",
		"<enderEmit><xLgr>RUA A</xLgr><nro>1</nro><xBairro>CENTRO</xBairro><cMun>4106902</cMun><xMun>CURITIBA</xMun><UF>PR</UF><CEP>80000000</CEP></enderEmit>", 1)
	nota, err := (&UploadNFEEntryUseCase{Repo: fiscalRepo, Docs: docs, Auth: auth}).ExecuteFile(ctx, []byte(s), request.UploadNFEDTO{})
	if err != nil {
		t.Fatal(err)
	}
	if nota.SupplierCode != nil || nota.PodeAprovar {
		t.Fatalf("emitente sem cadastro: fornecedor %v, pode aprovar %v", nota.SupplierCode, nota.PodeAprovar)
	}
	impedeFornecedor := false
	for _, p := range nota.Pendencias {
		if p.Campo == "supplier_code" && p.Nivel == "IMPEDE" {
			impedeFornecedor = true
		}
	}
	if !impedeFornecedor {
		t.Fatalf("faltou a pendência do fornecedor: %+v", nota.Pendencias)
	}

	// Item da nota → cadastro, com o almoxarifado e a unidade da nota.
	itens := &ItemDaNotaUseCase{Docs: docs, Cadastro: itemUC, Auth: auth}
	if _, err := itens.Execute(ctx, nota.ID, nota.Itens[0].ID, ItemDaNotaDTO{}); err == nil || !strings.Contains(err.Error(), "almoxarifado") {
		t.Fatalf("sem almoxarifado deveria pedir: %v", err)
	}
	chapa, err := itens.Execute(ctx, nota.ID, nota.Itens[0].ID, ItemDaNotaDTO{WarehouseID: &almox})
	if err != nil {
		t.Fatalf("cadastrando a chapa: %v", err)
	}
	luva, err := itens.Execute(ctx, nota.ID, nota.Itens[1].ID, ItemDaNotaDTO{WarehouseID: &almox, TipoUso: "CONSUMO"})
	if err != nil {
		t.Fatalf("cadastrando a luva: %v", err)
	}
	var nome, unid string
	var tipoUso int
	if err := pool.QueryRow(bg, `SELECT name, warehouse_unit_of_measurement::text, supplies_type_of_use FROM items WHERE code=$1 AND enterprise_id=$2`, luva.ItemCode, empresa).
		Scan(&nome, &unid, &tipoUso); err != nil {
		t.Fatal(err)
	}
	if nome != "LUVA DE RASPA CANO LONGO" || unid != "CX" || tipoUso != 1 || chapa.Unidade != "KG" {
		t.Fatalf("item da nota: %s %s uso %d; chapa %s", nome, unid, tipoUso, chapa.Unidade)
	}

	// Fornecedor da nota → cadastro, com o endereço do XML; a nota passa a ser dele.
	fornecedores := &FornecedorDaNotaUseCase{Docs: docs, Fiscal: fiscalRepo, Cadastro: supplierUC, Auth: auth}
	r, err := fornecedores.Execute(ctx, nota.ID, FornecedorDaNotaDTO{})
	if err != nil {
		t.Fatalf("cadastrando o fornecedor: %v", err)
	}
	if r.SupplierCode == nil {
		t.Fatalf("a nota não ficou com o fornecedor: %+v", r.Pendencias)
	}
	var docForn, ie, icms, rua, bairro, cep string
	if err := pool.QueryRow(bg, `SELECT s.document_number, COALESCE(s.state_registration,''), s.icms_contributor, COALESCE(a.street,''), COALESCE(a.neighborhood,''), COALESCE(a.zip_code,'')
		FROM suppliers s LEFT JOIN supplier_addresses a ON a.supplier_id = s.id WHERE s.code=$1 AND s.enterprise_id=$2`, *r.SupplierCode, empresa).
		Scan(&docForn, &ie, &icms, &rua, &bairro, &cep); err != nil {
		t.Fatal(err)
	}
	if docForn != cnpj || ie != "9012345678" || icms != "CONTRIBUINTE" || rua != "RUA A" || bairro != "CENTRO" || cep != "80000000" {
		t.Fatalf("fornecedor cadastrado: %s IE %s %s, endereço %s/%s/%s", docForn, ie, icms, rua, bairro, cep)
	}
	if _, err := fornecedores.Execute(ctx, nota.ID, FornecedorDaNotaDTO{}); err == nil {
		t.Error("cadastrar de novo deveria ser recusado (a nota já tem fornecedor)")
	}

	// Conciliação com os itens cadastrados: a nota fecha.
	doc, err := (&SaveFiscalEntryConciliationUseCase{Docs: docs, Fiscal: fiscalRepo, Auth: auth}).Execute(ctx, nota.ID, request.SaveFiscalEntryConciliationDTO{
		Itens: []request.FiscalEntryItemConciliationDTO{
			{ID: nota.Itens[0].ID, ItemCode: &chapa.ItemCode, PlanoContasID: &plano, WarehouseID: &almox, LembrarVinculo: true},
			{ID: nota.Itens[1].ID, ItemCode: &luva.ItemCode, PlanoContasID: &plano, WarehouseID: &almox},
		},
	})
	if err != nil {
		t.Fatalf("conciliação: %v", err)
	}
	if !doc.PodeAprovar {
		t.Fatalf("nota deveria fechar: %+v", doc.Pendencias)
	}

	// Fornecedor bloqueado trava a aprovação.
	testutil.Exec(t, pool, `UPDATE suppliers SET blocked=TRUE WHERE code=$1 AND enterprise_id=$2`, *r.SupplierCode, empresa)
	if _, err := (&ApproveFiscalEntryUseCase{FiscalRepo: fiscalRepo, Docs: docs, Auth: auth}).Execute(ctx, request.ApproveFiscalEntryDTO{ID: nota.ID}); err == nil || !strings.Contains(err.Error(), "bloqueado") {
		t.Fatalf("fornecedor bloqueado deveria impedir: %v", err)
	}
	testutil.Exec(t, pool, `UPDATE suppliers SET blocked=FALSE WHERE code=$1 AND enterprise_id=$2`, *r.SupplierCode, empresa)

	// Fornecedor cadastrado por fora (VSUP0500) depois da importação: a nota se liga sozinha.
	chave2 := fmt.Sprintf("41%042d", u+12)
	cnpj2 := cnpjComDV(fmt.Sprintf("%012d", (u+92)%1_000_000_000_000))
	s2 := strings.ReplaceAll(strings.ReplaceAll(string(base), "41261012345678000190550010000123451000123459", chave2), "12345678000190", cnpj2)
	nota2, err := (&UploadNFEEntryUseCase{Repo: fiscalRepo, Docs: docs, Auth: auth}).ExecuteFile(ctx, []byte(s2), request.UploadNFEDTO{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := supplierUC.CreateSupplier(ctx, request.CreateSupplierDTO{Name: "OUTRO", PersonType: "JURIDICA", DocumentType: "CNPJ", DocumentNumber: cnpj2,
		ICMSContributor: "NAO_CONTRIBUINTE"}); err != nil {
		t.Fatal(err)
	}
	visto, err := (&GetFiscalEntryUseCase{Repo: fiscalRepo, Docs: docs, Auth: auth}).Execute(ctx, nota2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if visto.SupplierCode == nil {
		t.Fatal("a nota deveria se ligar ao fornecedor cadastrado depois, pelo CNPJ")
	}
}

// cnpjComDV completa 12 dígitos com os dígitos verificadores (o cadastro de
// fornecedores recusa CNPJ inválido).
func cnpjComDV(base string) string {
	dv := func(s string, pesos []int) byte {
		soma := 0
		for i, p := range pesos {
			soma += int(s[i]-'0') * p
		}
		r := soma % 11
		if r < 2 {
			return '0'
		}
		return byte('0' + 11 - r)
	}
	p1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	s := base + string(dv(base, p1))
	return s + string(dv(s, append([]int{6}, p1...)))
}
