//go:build integration

package fiscal_test

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	repository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Unificação da VFIS0330 na VFIS0320 (migração 000433): a regra ganha
// fornecedor, que tem de ser da empresa, e regra com fornecedor (de entrada)
// não tributa a nota de saída. Percentual com 4 casas não é arredondado.
func TestRegraComFornecedorEPercentualDeQuatroCasas(t *testing.T) {
	pool := testutil.Pool(t)
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		testutil.Exec(t, pool, `DELETE FROM icms_ipi_tax_params WHERE enterprise_id = ANY($1)`, ids)
		testutil.Exec(t, pool, `DELETE FROM suppliers WHERE enterprise_id = ANY($1)`, ids)
	})
	fornecedor := 1_000_000_000 + testutil.UniqueCode()%100_000_000
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'Fornecedor A','44444444000191',$2,$3)`, fornecedor, ator, a.ID)
	repo := repository.NewFiscalParamsRepository(sqlc.New(pool), pool)

	// Fornecedor de outra empresa é recusado na inclusão e na alteração.
	if _, err := repo.CreateTaxParam(b.Ctx, &entity.ICMSIPITaxParam{UF: "PR", OperationType: entity.TaxParamOperationEntrada, SupplierCode: &fornecedor}); err == nil {
		t.Fatal("regra com fornecedor de outra empresa foi gravada")
	}
	ncm := "73269090"
	geral, err := repo.CreateTaxParam(b.Ctx, &entity.ICMSIPITaxParam{UF: "PR", OperationType: entity.TaxParamOperationAmbas, NCMCode: &ncm})
	if err != nil {
		t.Fatal(err)
	}
	geral.SupplierCode = &fornecedor
	if _, err := repo.UpdateTaxParam(b.Ctx, geral); err == nil {
		t.Fatal("alteração apontou para fornecedor de outra empresa")
	}

	// Na própria empresa grava, com o diferimento de 4 casas intacto.
	comFornecedor, err := repo.CreateTaxParam(a.Ctx, &entity.ICMSIPITaxParam{UF: "PR", OperationType: entity.TaxParamOperationAmbas,
		NCMCode: &ncm, SupplierCode: &fornecedor, ICMSDeferralPct: 33.3333, IsPreferred: true})
	if err != nil {
		t.Fatal(err)
	}
	if comFornecedor.SupplierCode == nil || *comFornecedor.SupplierCode != fornecedor || comFornecedor.ICMSDeferralPct != 33.3333 {
		t.Fatalf("regra gravada: fornecedor %v diferimento %v", comFornecedor.SupplierCode, comFornecedor.ICMSDeferralPct)
	}

	// Na saída a regra com fornecedor não vale — mesmo preferencial.
	r := repository.NewParametroICMSIPIPG(pool)
	if p, err := r.ParametroICMSIPI(a.Ctx, engine.ConsultaParametroICMSIPI{NCM: &ncm, UF: "PR"}); err != nil || p != nil {
		t.Fatalf("saída casou regra de fornecedor: %+v %v", p, err)
	}
	semFornecedor, err := repo.CreateTaxParam(a.Ctx, &entity.ICMSIPITaxParam{UF: "PR", OperationType: entity.TaxParamOperationSaida, NCMCode: &ncm, ICMSDeferralPct: 41.6667})
	if err != nil {
		t.Fatal(err)
	}
	p, err := r.ParametroICMSIPI(a.Ctx, engine.ConsultaParametroICMSIPI{NCM: &ncm, UF: "PR"})
	if err != nil || p == nil || p.ID != semFornecedor.ID || p.DiferimentoPct != 41.6667 {
		t.Fatalf("saída: %+v %v, quer a regra %d com 41,6667%%", p, err, semFornecedor.ID)
	}
}
