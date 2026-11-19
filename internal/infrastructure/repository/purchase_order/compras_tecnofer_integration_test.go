//go:build integration

package purchase_order

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Linha da tabela de preço gravada com fornecedor 0 é preço GENÉRICO. A
// Tecnofer tinha as 5 linhas da tabela com 0: o índice e a tela tratavam como
// genérico, a busca de preço do pedido não — a tabela inteira era ignorada e a
// aprovação recusava por falta de preço. Migração 000505: o 0 vira vazio.
func TestTabelaDeCompraComFornecedorZeroEhGenerica(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM purchase_price_tables WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM suppliers WHERE enterprise_id = ANY($1)`, ids)
	})
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, ator)
	testutil.Exec(t, pool, `UPDATE items SET warehouse_unit_of_measurement='UN', supplies_purchase_uom=NULL WHERE code=$1`, item)
	forn := 1_800_000_000 + testutil.UniqueCode()%100_000_000
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,$2,$3,$4,$5)`,
		forn, fmt.Sprintf("FORN %d", forn), fmt.Sprintf("%014d", forn), ator, a.ID)

	// Gravação direta com 0 (o que a tela/planilha antiga mandava): o gatilho grava vazio.
	var tabela, linha int64
	if err := pool.QueryRow(base, `INSERT INTO purchase_price_tables(code,description,created_by,enterprise_id,supplier_code)
		VALUES($1,'TAB ZERO',$2,$3,0) RETURNING id`, testutil.UniqueCode(), ator, a.ID).Scan(&tabela); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO purchase_price_table_items(table_id,item_code,supplier_code,uom,price) VALUES($1,$2,0,'UN',37.5) RETURNING id`,
		tabela, item).Scan(&linha); err != nil {
		t.Fatal(err)
	}
	var semFornecedor bool
	if err := pool.QueryRow(base, `SELECT (SELECT supplier_code IS NULL FROM purchase_price_tables WHERE id=$1) AND (SELECT supplier_code IS NULL FROM purchase_price_table_items WHERE id=$2)`,
		tabela, linha).Scan(&semFornecedor); err != nil {
		t.Fatal(err)
	}
	if !semFornecedor {
		t.Fatal("fornecedor 0 deveria ser gravado vazio (genérico) na tabela e na linha")
	}
	testutil.Exec(t, pool, `UPDATE purchase_price_table_items SET supplier_code=-1 WHERE id=$1`, linha)
	if err := pool.QueryRow(base, `SELECT supplier_code IS NULL FROM purchase_price_table_items WHERE id=$1`, linha).Scan(&semFornecedor); err != nil {
		t.Fatal(err)
	}
	if !semFornecedor {
		t.Fatal("alterar para fornecedor ≤ 0 deveria gravar vazio")
	}

	r := NewPurchaseOrderRepositorySQLC(pool)
	c, err := r.CondicaoDeCompra(a.Ctx, item, forn, "")
	if err != nil || len(c.Precos) != 1 || !c.Precos[0].Preco.Equal(decimal.RequireFromString("37.5")) {
		t.Fatalf("a linha genérica (antes gravada com 0) deveria dar o preço 37,5 para qualquer fornecedor: %+v %v", c, err)
	}
	// A empresa B não enxerga a tabela da A.
	if c, err := r.CondicaoDeCompra(b.Ctx, item, forn, ""); err == nil && len(c.Precos) > 0 {
		t.Fatalf("a empresa B leu o preço da tabela da A: %+v", c)
	}
}

// Pedido recusado não queima número: o número é reservado na transação que
// grava o pedido. A Tecnofer via a numeração pular (9, 10, 13, 14, 15).
func TestNumeroDoPedidoDeCompraSoAndaQuandoGrava(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, id := range []int64{a.ID, b.ID} {
			pool.Exec(base, `DELETE FROM purchase_order_items WHERE purchase_order_code IN (SELECT code FROM purchase_orders WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1))`, id)
			pool.Exec(base, `DELETE FROM purchase_orders WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`, id)
			pool.Exec(base, `DELETE FROM purchase_order_sequences WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`, id)
			pool.Exec(base, `DELETE FROM item_preferred_suppliers WHERE enterprise_id=$1`, id)
			pool.Exec(base, `DELETE FROM items WHERE enterprise_id=$1`, id)
			pool.Exec(base, `DELETE FROM suppliers WHERE enterprise_id=$1`, id)
		}
	})
	r := NewPurchaseOrderRepositorySQLC(pool)
	fornecedores := map[int64]int64{}
	for _, e := range []testutil.Empresa{a, b} {
		forn := 1_810_000_000 + testutil.UniqueCode()%100_000_000
		testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,$2,$3,$4,$5)`,
			forn, fmt.Sprintf("FORN %d", forn), fmt.Sprintf("%014d", forn), ator, e.ID)
		fornecedores[e.ID] = forn
	}
	pedido := func(e testutil.Empresa) *entity.PurchaseOrder {
		forn := fornecedores[e.ID]
		return &entity.PurchaseOrder{EnterpriseCode: e.Code, Status: entity.PurchaseOrderStatusDRAFT, Origin: entity.PurchaseOrderOriginNORMAL,
			EmissionDate: time.Now(), SupplierCode: &forn, CurrencyCode: "BRL", CreatedBy: ator}
	}
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, ator)
	testutil.Exec(t, pool, `INSERT INTO item_preferred_suppliers(enterprise_id,item_code,supplier_code,supplier_item_code,created_by) VALUES($1,$2,$3,'X',$4)`,
		a.ID, item, fornecedores[a.ID], ator)
	linha := func(code int64) []*entity.PurchaseOrderItem {
		return []*entity.PurchaseOrderItem{{Sequence: 1, ItemCode: code, RequestedQty: decimal.NewFromInt(2), UnitPrice: decimal.NewFromInt(10),
			Status: entity.PurchaseOrderItemStatusOPEN, IsActive: true}}
	}

	p1, err := r.CreateWithItems(a.Ctx, pedido(a), linha(item))
	if err != nil {
		t.Fatal(err)
	}
	// Linha com item sem fonte de suprimento: o banco recusa a linha e o
	// pedido inteiro volta — com o número.
	semFonte := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, semFonte, ator)
	if _, err := r.CreateWithItems(a.Ctx, pedido(a), linha(semFonte)); err == nil {
		t.Fatal("pedido com item sem fonte de suprimento deveria ser recusado")
	}
	p2, err := r.Create(a.Ctx, pedido(a))
	if err != nil {
		t.Fatal(err)
	}
	if p2.OrderNumber != p1.OrderNumber+1 {
		t.Fatalf("a recusa queimou número: %d e depois %d", p1.OrderNumber, p2.OrderNumber)
	}
	// Numeração por empresa: a B começa a sua.
	pb, err := r.Create(b.Ctx, pedido(b))
	if err != nil {
		t.Fatal(err)
	}
	if pb.OrderNumber != 1 {
		t.Fatalf("a empresa B deveria começar no 1, veio %d", pb.OrderNumber)
	}
}
