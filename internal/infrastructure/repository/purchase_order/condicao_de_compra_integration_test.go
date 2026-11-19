//go:build integration

package purchase_order

import (
	"context"
	"fmt"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A sugestão do MRP sai na unidade de compra do fornecedor e com o preço da
// tabela vigente dele — da empresa da sessão, nunca da outra.
func TestCondicaoDeCompraDuasEmpresas(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM purchase_price_tables WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM item_preferred_suppliers WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM suppliers WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM units_of_measure WHERE enterprise_id = ANY($1) AND NOT is_system`, ids)
	})
	for _, e := range []testutil.Empresa{a, b} {
		for _, u := range []struct {
			code, dim string
			casas     int
		}{{"UN", "CONTAGEM", 0}, {"KG", "MASSA", 4}, {"CHP", "CONTAGEM", 0}} {
			testutil.Exec(t, pool, `INSERT INTO units_of_measure(enterprise_id,code,description,dimension,fiscal_code,decimals)
				VALUES($1,$2,$2,$3,$2,$4) ON CONFLICT (enterprise_id, code) DO NOTHING`, e.ID, u.code, u.dim, u.casas)
		}
	}
	chapa := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, chapa, ator)
	testutil.Exec(t, pool, `UPDATE items SET warehouse_unit_of_measurement='KG', supplies_purchase_uom='UN' WHERE code=$1`, chapa)
	forn := func(e testutil.Empresa) int64 {
		c := 1_700_000_000 + testutil.UniqueCode()%100_000_000
		testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,$2,$3,$4,$5)`,
			c, fmt.Sprintf("FORN %d", c), fmt.Sprintf("%014d", c), ator, e.ID)
		return c
	}
	fornecedor, outro := forn(a), forn(a)
	tabela := func(e testutil.Empresa, supplier *int64, uom string, fim string, faixas ...[2]string) {
		t.Helper()
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO purchase_price_tables(code,description,created_by,enterprise_id,supplier_code,validity_end)
			VALUES($1,'TAB',$2,$3,$4,CASE WHEN $5='' THEN NULL ELSE CURRENT_DATE + $5::int END) RETURNING id`,
			testutil.UniqueCode(), ator, e.ID, supplier, fim).Scan(&id); err != nil {
			t.Fatal(err)
		}
		for _, f := range faixas {
			testutil.Exec(t, pool, `INSERT INTO purchase_price_table_items(table_id,item_code,uom,price,min_qty) VALUES($1,$2,NULLIF($3,''),$4,$5)`, id, chapa, uom, f[0], f[1])
		}
	}
	r := NewPurchaseOrderRepositorySQLC(pool)
	precos := func(c purchase_order_uc.CondicaoDeCompra) string {
		out := ""
		for _, p := range c.Precos {
			out += fmt.Sprintf("%s/%s≥%s ", p.Preco.String(), p.Unidade, p.MinQtd.String())
		}
		return out
	}

	// Sem tabela: unidade de compra do item (UN, 0 casas), sem preço.
	c, err := r.CondicaoDeCompra(a.Ctx, chapa, fornecedor, "")
	if err != nil || c.Unidade != "UN" || c.Casas != 0 || len(c.Precos) != 0 {
		t.Fatalf("sem tabela: %+v %v", c, err)
	}

	// Tabela geral, duas do fornecedor (a de volume a partir de 500 kg), de outro fornecedor e uma
	// do fornecedor já vencida: vêm as do fornecedor (faixa maior primeiro),
	// depois a geral; a de outro fornecedor e a vencida ficam de fora.
	tabela(a, nil, "KG", "", [2]string{"11", "0"})
	tabela(a, &fornecedor, "KG", "", [2]string{"12.5", "0"})
	tabela(a, &fornecedor, "KG", "", [2]string{"10.9", "500"}) // tabela de volume: a partir de 500 kg
	tabela(a, &outro, "KG", "", [2]string{"99", "0"})
	tabela(a, &fornecedor, "KG", "-5", [2]string{"1", "0"})
	c, err = r.CondicaoDeCompra(a.Ctx, chapa, fornecedor, "")
	if err != nil || precos(c) != "10.9/KG≥500 12.5/KG≥0 11/KG≥0 " {
		t.Fatalf("preços do fornecedor: %s %v", precos(c), err)
	}

	// Desconto em cadeia na tabela do fornecedor (VSUP0120): 10 % e depois R$ 0,25
	// de acréscimo → 12,5 × 0,9 + 0,25 = 11,50. Vale o líquido.
	testutil.Exec(t, pool, `INSERT INTO purchase_price_item_adjustments(price_item_id,sequence,adjustment_kind,calculation_type,value)
		SELECT i.id, s.seq, s.kind, s.calc, s.v FROM purchase_price_table_items i
		  JOIN purchase_price_tables t ON t.id = i.table_id,
		  (VALUES (1,'DISCOUNT','PERCENT',10::numeric),(2,'SURCHARGE','FIXED',0.25)) s(seq,kind,calc,v)
		 WHERE t.enterprise_id = $1 AND t.supplier_code = $2 AND i.item_code = $3 AND i.price = 12.5`, a.ID, fornecedor, chapa)
	if c, err = r.CondicaoDeCompra(a.Ctx, chapa, fornecedor, ""); err != nil || precos(c) != "10.9/KG≥500 11.5/KG≥0 11/KG≥0 " {
		t.Fatalf("preço líquido: %s %v", precos(c), err)
	}

	// Fonte de suprimento do fornecedor em chapa (CHP) vence a unidade do item.
	testutil.Exec(t, pool, `INSERT INTO item_preferred_suppliers(item_code,supplier_code,uom,created_by,enterprise_id) VALUES($1,$2,'CHP',$3,$4)`,
		chapa, fornecedor, ator, a.ID)
	if c, err = r.CondicaoDeCompra(a.Ctx, chapa, fornecedor, ""); err != nil || c.Unidade != "CHP" {
		t.Fatalf("unidade da fonte do fornecedor: %+v %v", c, err)
	}

	// A empresa B não lê o item nem a tabela da A.
	if _, err := r.CondicaoDeCompra(b.Ctx, chapa, fornecedor, ""); err == nil {
		t.Fatal("empresa B leu a condição de compra do item da A")
	}
	// Nem uma tabela da B entra no preço da A.
	testutil.SeedItem(t, pool, b.Ctx, chapa+1, ator)
	tabela(b, nil, "", "", [2]string{"1", "0"})
	if c, _ = r.CondicaoDeCompra(a.Ctx, chapa, fornecedor, ""); precos(c) != "10.9/KG≥500 11.5/KG≥0 11/KG≥0 " {
		t.Fatalf("preço misturado entre empresas: %s", precos(c))
	}
}
