//go:build integration

package itemgovernanca

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/items/governanca"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A liberação confere TODAS as unidades em que o item é comprado. A Tecnofer
// tinha chapa com "CHAPA" no cadastro (com conversão CHAPA→kg) e "UN" na fonte
// de suprimento: a liberação via só a do cadastro e dava o item como pronto;
// o pedido sai na unidade da fonte e era recusado na aprovação.
func TestLiberacaoConfereAUnidadeDaFonteDeSuprimento(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, id := range []int64{a.ID, b.ID} {
			pool.Exec(base, `DELETE FROM item_unit_conversions WHERE item_code IN (SELECT code FROM items WHERE enterprise_id=$1)`, id)
			pool.Exec(base, `DELETE FROM item_preferred_suppliers WHERE enterprise_id=$1`, id)
			pool.Exec(base, `DELETE FROM purchase_price_tables WHERE enterprise_id=$1`, id)
			pool.Exec(base, `DELETE FROM items WHERE enterprise_id=$1`, id)
			pool.Exec(base, `DELETE FROM suppliers WHERE enterprise_id=$1`, id)
			pool.Exec(base, `DELETE FROM units_of_measure WHERE enterprise_id=$1 AND NOT is_system`, id)
		}
	})
	for _, e := range []testutil.Empresa{a, b} {
		for _, u := range [][2]string{{"UN", "CONTAGEM"}, {"KG", "MASSA"}, {"CHAPA", "CONTAGEM"}} {
			testutil.Exec(t, pool, `INSERT INTO units_of_measure(enterprise_id,code,description,dimension,fiscal_code,decimals)
				VALUES($1,$2,$2,$3,$2,0) ON CONFLICT (enterprise_id, code) DO NOTHING`, e.ID, u[0], u[1])
		}
	}
	chapa := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, chapa, ator)
	testutil.Exec(t, pool, `UPDATE items SET warehouse_unit_of_measurement='KG', supplies_purchase_uom='CHAPA', engineering_type=1 WHERE code=$1`, chapa)
	testutil.Exec(t, pool, `INSERT INTO item_unit_conversions(item_code,from_uom,to_uom,factor,created_by) VALUES($1,'CHAPA','KG',62.3,$2)`, chapa, ator)
	forn := 1_820_000_000 + testutil.UniqueCode()%100_000_000
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,$2,$3,$4,$5)`,
		forn, fmt.Sprintf("FORN %d", forn), fmt.Sprintf("%014d", forn), ator, a.ID)
	r := New(pool)
	conferir := func(ctx context.Context) governanca.Ficha {
		t.Helper()
		f, err := r.Ficha(ctx, chapa)
		if err != nil {
			t.Fatal(err)
		}
		return *f
	}

	// Só o cadastro (CHAPA, com conversão): pronto para comprar.
	if f := conferir(a.Ctx); !f.CompraEmOutraUM || !f.TemConversao {
		t.Fatalf("CHAPA→KG cadastrada: deveria estar pronto (%+v %v)", f.CompraEmOutraUM, f.UnidadesSemConversao)
	}
	// Fonte de suprimento em UN, sem conversão UN→KG: pendente, dizendo onde.
	testutil.Exec(t, pool, `INSERT INTO item_preferred_suppliers(enterprise_id,item_code,supplier_code,supplier_item_code,uom,created_by) VALUES($1,$2,$3,'CH-794','UN',$4)`,
		a.ID, chapa, forn, ator)
	f := conferir(a.Ctx)
	if f.TemConversao || len(f.UnidadesSemConversao) != 1 || !strings.Contains(f.UnidadesSemConversao[0], "UN→KG (fonte de suprimento)") {
		t.Fatalf("UN da fonte sem conversão deveria ficar pendente: %v", f.UnidadesSemConversao)
	}
	texto := fmt.Sprint(governanca.AvaliarProntidao(f, "LIBERADO", nil))
	if !strings.Contains(texto, "UN→KG (fonte de suprimento)") {
		t.Fatalf("a pendência de compra deveria dizer qual unidade falta: %s", texto)
	}
	// Linha de tabela de preço vigente em UN: aparece também.
	var tab int64
	if err := pool.QueryRow(base, `INSERT INTO purchase_price_tables(code,description,created_by,enterprise_id) VALUES($1,'TAB',$2,$3) RETURNING id`,
		testutil.UniqueCode(), ator, a.ID).Scan(&tab); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `UPDATE item_preferred_suppliers SET uom='CHAPA' WHERE item_code=$1`, chapa)
	testutil.Exec(t, pool, `INSERT INTO purchase_price_table_items(table_id,item_code,uom,price) VALUES($1,$2,'UN',10)`, tab, chapa)
	if f := conferir(a.Ctx); f.TemConversao || len(f.UnidadesSemConversao) != 1 || !strings.Contains(f.UnidadesSemConversao[0], "tabela de preço") {
		t.Fatalf("UN da tabela de preço sem conversão deveria ficar pendente: %v", f.UnidadesSemConversao)
	}
	// Cadastrada a conversão UN→KG: pronto.
	testutil.Exec(t, pool, `INSERT INTO item_unit_conversions(item_code,from_uom,to_uom,factor,created_by) VALUES($1,'UN','KG',62.3,$2)`, chapa, ator)
	if f := conferir(a.Ctx); !f.TemConversao || len(f.UnidadesSemConversao) != 0 {
		t.Fatalf("com UN→KG cadastrada deveria estar pronto: %v", f.UnidadesSemConversao)
	}
	// A empresa B não lê a ficha do item da A.
	if _, err := r.Ficha(b.Ctx, chapa); err == nil {
		t.Fatal("a empresa B leu a ficha do item da A")
	}
}
