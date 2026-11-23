//go:build integration

package custo_industrial_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/custo_industrial"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A margem do pedido lê o pedido da empresa (enterprise_code) e o custo-padrão
// do item da empresa; a B não lê o pedido da A.
func TestMargemDoPedidoPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	item := testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, item)
	testutil.Exec(t, pool, `INSERT INTO item_standard_costs(item_code,mask,material_cost,labor_cost,overhead_cost,calculated_at,calculated_by) VALUES($1,'',230.22,0,0,NOW(),$2)`, item, ator)
	var pedido int64
	if err := pool.QueryRow(base, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by) VALUES($1,$2,'P',$3,$4) RETURNING code`,
		testutil.UniqueCode()%1_000_000, a.Code, testutil.UniqueCode(), ator).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,total_net) VALUES($1,1,$2,1,360.50,360.50)`, pedido, item)
	t.Cleanup(func() {
		pool.Exec(base, `DELETE FROM sales_order_items WHERE sales_order_code=$1`, pedido)
		pool.Exec(base, `DELETE FROM sales_orders WHERE code=$1`, pedido)
		pool.Exec(base, `DELETE FROM item_standard_costs WHERE item_code=$1`, item)
	})
	repo := custo_industrial.New(pool)
	l, err := repo.LinhasDoPedido(a.Ctx, pedido)
	if err != nil {
		t.Fatal(err)
	}
	if len(l) != 1 || !l[0].Receita.Equal(decimal.RequireFromString("360.5")) || !l[0].CustoPadraoUnit.Equal(decimal.RequireFromString("230.22")) || !l[0].TemPadrao {
		t.Fatalf("linhas = %+v", l)
	}
	if lb, err := repo.LinhasDoPedido(b.Ctx, pedido); err != nil || len(lb) != 0 {
		t.Fatalf("a empresa B leu o pedido da A: %+v err=%v", lb, err)
	}
}
