//go:build integration

package impressao_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/impressao"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O PDF do plano de materiais mostrava o pedido só digitado (que o MRP não lê)
// e, por GREATEST(saldo, pedido), sempre a quantidade cheia do pedido — mesmo
// com parte já entregue. Agora: o mesmo filtro do MRP e o saldo.
func TestPlanoDeMateriaisMostraSoADemandaQueOMRPLeu(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(bg, `DELETE FROM mrp_planned_suggestions WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(bg, `DELETE FROM sales_order_items WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE enterprise_code = ANY($1))`, []int64{a.Code, b.Code})
		pool.Exec(bg, `DELETE FROM sales_orders WHERE enterprise_code = ANY($1)`, []int64{a.Code, b.Code})
		pool.Exec(bg, `DELETE FROM production_plans WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(bg, `DELETE FROM customers WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(bg, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	u := testutil.UniqueCode()
	item := u
	testutil.SeedItem(t, pool, a.Ctx, item, ator)
	plano := 70_000_000 + u%10_000_000
	cliente := 890_000 + u%90_000
	for _, e := range []testutil.Empresa{a, b} {
		testutil.Exec(t, pool, `INSERT INTO production_plans(code,name,created_by,enterprise_id) VALUES($1,'PLANO',$2,$3)`, plano+e.ID, ator, e.ID)
		testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE','11444777000161',$2,$3)`, cliente, ator, e.ID)
	}
	testutil.Exec(t, pool, `INSERT INTO mrp_planned_suggestions(plan_code,item_code,quantity,need_date,order_type,demand_type,enterprise_id)
		VALUES($1,$2,10,CURRENT_DATE+10,'FABRICACAO','SALES_ORDER',$3)`, plano+a.ID, item, a.ID)
	pedido := func(e testutil.Empresa, status string, pedida, atendida int) int64 {
		t.Helper()
		var code int64
		if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,customer_code,created_by,status) VALUES($1,$2,$3,$4,$5) RETURNING code`,
			testutil.UniqueCode()%900_000, e.Code, cliente, ator, status).Scan(&code); err != nil {
			t.Fatal(err)
		}
		st := "OPEN"
		if atendida > 0 {
			st = "PARTIAL"
		}
		testutil.Exec(t, pool, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,attended_qty,unit_price,total_net,status)
			VALUES($1,1,$2,$3,$4,10,0,$5)`, code, item, pedida, atendida, st)
		return code
	}
	pedido(a, "R", 50, 0) // só digitado: o MRP não lê
	pedido(a, "P", 30, 12)
	pedido(b, "P", 99, 0) // da outra empresa

	c := impressao.New(pool)
	p, err := c.PlanoMateriais(bg, uc.Tenant{ID: a.ID, Code: a.Code}, plano+a.ID)
	if err != nil {
		t.Fatal(err)
	}
	var vendas []uc.DemandaPlano
	for _, d := range p.Demandas {
		if d.Origem == "Pedido de venda" {
			vendas = append(vendas, d)
		}
	}
	if len(vendas) != 1 {
		t.Fatalf("demandas de venda no plano: %d (%+v), esperado só o pedido liberado", len(vendas), vendas)
	}
	if !vendas[0].Quantidade.Equal(decimal.NewFromInt(18)) {
		t.Fatalf("quantidade %s, esperado o saldo 18 (30 − 12 entregues)", vendas[0].Quantidade)
	}
}
