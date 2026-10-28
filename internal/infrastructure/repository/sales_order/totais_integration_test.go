//go:build integration

package sales_order_test

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	salesorder "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A capa do pedido fecha com a soma das linhas ativas (cancelada não conta), e
// a empresa B não recalcula — nem zera — o pedido da A.
func TestRecalculaTotaisDoPedido(t *testing.T) {
	pool := testutil.Pool(t)
	ator := testutil.Actor(t, pool)
	var pedido int64
	a, b := testutil.DuasEmpresas(t, pool, func(a, _ testutil.Empresa) {
		testutil.Exec(t, pool, `DELETE FROM sales_order_items WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE enterprise_code=$1)`, a.Code)
		testutil.Exec(t, pool, `DELETE FROM sales_orders WHERE enterprise_code=$1`, a.Code)
	})
	if err := pool.QueryRow(a.Ctx, `INSERT INTO sales_orders(order_number,enterprise_code,created_by) VALUES(1,$1,$2) RETURNING code`, a.Code, ator).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	item := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO sales_order_items(sales_order_code,item_code,total_net,total_ipi,total_st,status,is_active) VALUES
		($1,$2,1000,50,30,'OPEN',true),($1,$2,500,25,0,'OPEN',true),($1,$2,9999,99,99,'CANCELLED',false)`, pedido, item)
	repo := salesorder.NewSalesOrderRepositorySQLC(sqlc.New(pool))

	// Outra empresa não alcança o pedido.
	if err := repo.RecalculateTotals(b.Ctx, pedido); err != nil {
		t.Fatal(err)
	}
	var bruto, semST, comST decimal.Decimal
	ler := func() {
		t.Helper()
		if err := pool.QueryRow(a.Ctx, `SELECT total_gross, total_net_no_st, total_with_ipi_with_st FROM sales_orders WHERE code=$1`, pedido).Scan(&bruto, &semST, &comST); err != nil {
			t.Fatal(err)
		}
	}
	ler()
	if !bruto.IsZero() {
		t.Fatalf("a empresa B recalculou o pedido da A: %s", bruto)
	}

	if err := repo.RecalculateTotals(a.Ctx, pedido); err != nil {
		t.Fatal(err)
	}
	ler()
	if !bruto.Equal(decimal.NewFromInt(1500)) || !semST.Equal(decimal.NewFromInt(1575)) || !comST.Equal(decimal.NewFromInt(1605)) {
		t.Fatalf("totais: bruto %s, sem ST %s, com IPI e ST %s; quer 1500, 1575 e 1605", bruto, semST, comST)
	}
}
