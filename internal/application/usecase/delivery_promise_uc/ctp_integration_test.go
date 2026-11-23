//go:build integration

package delivery_promise_uc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/service"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	dprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_promise"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	stockpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Melhoria 9: "e se entrar um pedido de 50?" — 10 em estoque, 40 chegando num
// pedido de compra e 30 já pedidos por outro cliente: 8 sai do estoque, a
// segunda linha de 20 vem da compra e 50 só com compra nova (prazo do item).
func TestCenarioDePromessa(t *testing.T) {
	q, pool := testutil.Queries(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		pool.Exec(base, `DELETE FROM stock_movements WHERE enterprise_id=$1`, a.ID)
		pool.Exec(base, `DELETE FROM stock_balances WHERE enterprise_id=$1`, a.ID)
	})
	item := testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, item)
	testutil.Exec(t, pool, `UPDATE items SET engineering_type=1, planning_purchase_lead_time_days=10 WHERE code=$1`, item)
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	if _, err := stockpg.NewStockRepositorySQLC(pool).CreateMovement(a.Ctx, &stockentity.StockMovement{ItemCode: item, WarehouseID: almox, MovementType: "IN",
		Quantity: decimal.NewFromInt(10), UnitPrice: decimal.NewFromInt(1), CreatedBy: ator}); err != nil {
		t.Fatal(err)
	}
	fornecedor := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_type,document_number,created_by,enterprise_id) VALUES($1,'Aço Sul','CNPJ',$2,$3,$4)`, fornecedor, fmt.Sprint(fornecedor), ator, a.ID)
	t.Cleanup(func() { pool.Exec(base, `DELETE FROM suppliers WHERE code=$1`, fornecedor) })
	testutil.FonteDeSuprimento(t, pool, item, fornecedor)
	chega := time.Now().AddDate(0, 0, 5)
	var pc, pv int64
	if err := pool.QueryRow(base, `INSERT INTO purchase_orders(order_number,enterprise_code,status,supplier_code,created_by) VALUES($4,$1,'APPROVED',$2,$3) RETURNING code`, a.Code, fornecedor, ator, testutil.UniqueCode()%1_000_000).Scan(&pc); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO purchase_order_items(purchase_order_code,item_code,requested_qty,unit_price,delivery_date) VALUES($1,$2,40,1,$3::date)`, pc, item, chega)
	if err := pool.QueryRow(base, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by) VALUES($1,$2,'P',$3,$4) RETURNING code`,
		testutil.UniqueCode()%1_000_000, a.Code, testutil.UniqueCode(), ator).Scan(&pv); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,delivery_date) VALUES($1,1,$2,30,1,$3::date)`, pv, item, chega.AddDate(0, 0, 5))
	t.Cleanup(func() {
		pool.Exec(base, `DELETE FROM purchase_order_items WHERE purchase_order_code=$1`, pc)
		pool.Exec(base, `DELETE FROM purchase_orders WHERE code=$1`, pc)
		pool.Exec(base, `DELETE FROM sales_order_items WHERE sales_order_code=$1`, pv)
		pool.Exec(base, `DELETE FROM sales_orders WHERE code=$1`, pv)
	})
	uc := &delivery_promise_uc.PromessaUseCase{Repo: dprepo.New(pool), Items: itemrepo.NewRepositoryItemSQLC(q).ComPool(pool)}
	cod := request.TextCode(fmt.Sprint(item))
	if _, err := uc.Simular(b.Ctx, delivery_promise_uc.CenarioDTO{Lines: []delivery_promise_uc.LinhaDoCenario{{ItemCode: cod, Quantity: decimal.NewFromInt(1)}}}); err == nil {
		t.Fatal("a empresa B simulou com o item da A")
	}
	res, err := uc.Simular(a.Ctx, delivery_promise_uc.CenarioDTO{Lines: []delivery_promise_uc.LinhaDoCenario{
		{ItemCode: cod, Quantity: decimal.NewFromInt(8)},
		{ItemCode: cod, Quantity: decimal.NewFromInt(12)},
		{ItemCode: cod, Quantity: decimal.NewFromInt(50)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Origem != service.OrigemEstoque || res[1].Origem != service.OrigemEmAndamento || res[2].Origem != service.OrigemNovaCompra {
		t.Fatalf("origens = %s, %s, %s", res[0].Origem, res[1].Origem, res[2].Origem)
	}
	if res[1].Data.Format("2006-01-02") != chega.Format("2006-01-02") {
		t.Fatalf("a segunda linha vem com a compra de %s: %v", chega.Format("02/01"), res[1].Data)
	}
}
