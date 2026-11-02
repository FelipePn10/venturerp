//go:build integration

package fiscal_uc_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O código digitado na nota manual fica memorizado: a próxima nota do fornecedor já
// chega no nosso item, e o preço da nota aprovada vira a referência da próxima compra.
func TestIntegration_VinculoDaNotaManualEUltimoPreco(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := testutil.TenantContext(t, pool)
	actor := testutil.Actor(t, pool)
	empresa := testutil.EnterpriseID(t, ctx)
	docs := fiscalpg.NewFiscalEntryDocumentRepositoryPG(fiscalpg.NewFiscalRepositoryPG(pool))

	u := testutil.UniqueCode()
	item := 800_000_000 + u%100_000_000
	testutil.SeedItem(t, pool, ctx, item, actor)
	fornecedor := 700_000_000 + u%100_000_000
	codigo := fmt.Sprintf("FX-%d", u)
	hoje := time.Now()
	descricao := "CHAPA DO FORNECEDOR"
	uom := "UN"
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM fiscal_entries WHERE supplier_code = $1`, fornecedor)
		_, _ = pool.Exec(ctx, `DELETE FROM item_preferred_suppliers WHERE supplier_code = $1`, fornecedor)
	})

	nota := func(numero int64, status entity.FiscalEntryStatus) *entity.FiscalEntry {
		return &entity.FiscalEntry{
			EnterpriseID: empresa, NumeroNF: numero, Serie: "1", Modelo: "55", DataEmissao: hoje, DataEntrada: hoje,
			CnpjEmitente: fmt.Sprintf("%014d", u%100_000_000_000_000), RazaoSocialEmitente: "EMITENTE", SupplierCode: &fornecedor,
			Status: status, CreatedBy: actor,
			Itens: []*entity.FiscalEntryItem{{
				Sequence: 1, ItemCode: &item, SupplierItemCode: &codigo, LembrarVinculo: true, Description: &descricao, UOM: &uom,
				Cfop: "1101", Quantity: 2, UnitPrice: 42.5, TotalPrice: 85, ValorContabil: decimal.NewFromInt(85), MovimentaEstoque: true, GeraFinanceiro: true,
			}},
		}
	}
	if _, err := docs.CreateEntryDocument(ctx, nota(u%1_000_000+1, entity.EntryStatusApproved)); err != nil {
		t.Fatal(err)
	}

	vinculos, err := docs.SupplierLinksForEntry(ctx, fornecedor)
	if err != nil {
		t.Fatal(err)
	}
	if len(vinculos) != 1 || vinculos[0].Code != item || vinculos[0].SupplierItemCode != codigo {
		t.Fatalf("vínculo não memorizado: %+v", vinculos)
	}
	if vinculos[0].XMLUOM != nil {
		t.Fatalf("a unidade da nota manual não pode virar a unidade do fator de conversão: %s", *vinculos[0].XMLUOM)
	}

	// A segunda nota com o mesmo código não duplica o vínculo.
	if _, err := docs.CreateEntryDocument(ctx, nota(u%1_000_000+2, entity.EntryStatusPending)); err != nil {
		t.Fatal(err)
	}
	if vinculos, err = docs.SupplierLinksForEntry(ctx, fornecedor); err != nil || len(vinculos) != 1 {
		t.Fatalf("vínculo duplicado: %d (err=%v)", len(vinculos), err)
	}

	preco, err := docs.UltimoPrecoDeCompra(ctx, &fornecedor, item)
	if err != nil {
		t.Fatal(err)
	}
	if preco == nil || !preco.Preco.Equal(decimal.NewFromFloat(42.5)) || preco.DoPedido {
		t.Fatalf("último preço: %+v", preco)
	}

	// Outra empresa não vê o preço nem o vínculo.
	vizinha, _ := testutil.DuasEmpresas(t, pool, nil)
	outra := vizinha.Ctx
	if p, err := docs.UltimoPrecoDeCompra(outra, &fornecedor, item); err != nil || p != nil {
		t.Fatalf("preço vazou entre empresas: %+v (err=%v)", p, err)
	}
	if v, err := docs.SupplierLinksForEntry(outra, fornecedor); err != nil || len(v) != 0 {
		t.Fatalf("vínculo vazou entre empresas: %+v (err=%v)", v, err)
	}
}
