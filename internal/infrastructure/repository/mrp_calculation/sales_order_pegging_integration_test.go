//go:build integration

package mrp_calculation

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O atendimento do pedido de venda acha só os planos reais da empresa do
// pedido — cenário de simulação e outra empresa ficam de fora.
func TestSalesOrderPeggingPlansPorEmpresaSemCenario(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM mrp_pegging WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM production_plans WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM sales_order_items WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE enterprise_code = ANY($1))`, []int64{a.Code, b.Code})
		pool.Exec(base, `DELETE FROM sales_orders WHERE enterprise_code = ANY($1)`, []int64{a.Code, b.Code})
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, actor)
	var pedido, linha int64
	if err := pool.QueryRow(base, `INSERT INTO sales_orders(order_number,enterprise_code,created_by) VALUES($1,$2,$3) RETURNING code`,
		testutil.UniqueCode(), a.Code, actor).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO sales_order_items(sales_order_code,item_code) VALUES($1,$2) RETURNING code`, pedido, item).Scan(&linha); err != nil {
		t.Fatal(err)
	}
	real, cenario := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO production_plans(code,name,created_by,enterprise_id,is_scenario) VALUES($1,'Real',$3,$4,false),($2,'Cenário',$3,$4,true)`, real, cenario, actor, a.ID)
	dia := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	for _, p := range []int64{real, cenario} {
		testutil.Exec(t, pool, `INSERT INTO mrp_pegging(plan_code,item_code,demand_kind,demand_code,need_date,supply_kind,quantity,enterprise_id)
			VALUES($1,$2,$3,$4,$5::date,'ESTOQUE',2,$6)`, p, item, entity.PegDemandSalesOrder, linha, dia, a.ID)
	}
	repo := NewMRPCalculationRepositorySQLC(sqlc.New(pool), pool)

	planos, err := repo.SalesOrderPeggingPlans(a.Ctx, pedido)
	if err != nil {
		t.Fatal(err)
	}
	if len(planos) != 1 || planos[0].PlanCode != real || len(planos[0].ItemCodes) != 1 || planos[0].ItemCodes[0] != linha {
		t.Fatalf("planos do pedido = %+v; esperado só o plano real %d com a linha %d", planos, real, linha)
	}
	if outra, err := repo.SalesOrderPeggingPlans(b.Ctx, pedido); err != nil || len(outra) != 0 {
		t.Fatalf("a empresa B enxerga o atendimento do pedido da A: %+v (%v)", outra, err)
	}
}
