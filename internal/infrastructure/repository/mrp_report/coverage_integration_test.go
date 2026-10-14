//go:build integration

package mrp_report_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_report_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_report"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A cobertura lê o perfil do último cálculo do plano, o estoque e as ordens
// planejadas ainda não firmadas — só da empresa da sessão.
func TestCoberturaLePerfilEstoqueEPlanejadasDaEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM planned_orders WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM mrp_planned_suggestions WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM mrp_item_profiles WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM stock_balances WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM production_plans WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, actor)
	itemB := testutil.UniqueCode()
	testutil.SeedItem(t, pool, b.Ctx, itemB, actor)
	plano := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO production_plans(code,name,created_by,enterprise_id) VALUES($1,'A',$2,$3)`, plano, actor, a.ID)
	hoje := time.Now()
	d1, d2 := hoje.AddDate(0, 0, 5), hoje.AddDate(0, 0, 12)
	for e, codigo := range map[int64]int64{a.ID: item, b.ID: itemB} {
		testutil.Exec(t, pool, `INSERT INTO mrp_item_profiles(item_code,plan_code,calculation_date,demand,orders_planned,orders_firm,need_date,enterprise_id)
			VALUES($1,$2,$3::date,15,0,10,$4::date,$6),($1,$2,$3::date,20,25,0,$5::date,$6)`, codigo, plano, hoje, d1, d2, e)
		testutil.Exec(t, pool, `INSERT INTO stock_balances(item_code,warehouse_id,quantity,reserved_qty,enterprise_id) VALUES($1,1,12,2,$2)`, codigo, e)
	}
	var firmada int64
	if err := pool.QueryRow(base, `INSERT INTO mrp_planned_suggestions(plan_code,item_code,quantity,need_date,start_date,order_type,enterprise_id)
		VALUES($1,$2,25,$3::date,$4::date,'FABRICACAO',$5),($1,$2,5,$3::date,$3::date,'FABRICACAO',$5) RETURNING code`, plano, item, d2, d1, a.ID).Scan(&firmada); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,status,demand_type,demand_code,need_date,created_by,enterprise_id,is_active,mrp_suggestion_code)
		VALUES($1,$2,25,'PRODUCTION','PLANNED','INDEPENDENT',0,$3::date,$4,$5,true,$6)`, testutil.UniqueCode(), item, d2, actor, a.ID, firmada)

	repo := mrp_report.New(pool)
	lido, err := repo.Coverage(a.Ctx, mrp_report_uc.Filter{PlanCode: &plano})
	if err != nil {
		t.Fatal(err)
	}
	if len(lido) != 1 {
		t.Fatalf("itens = %d, esperado 1 (só o da empresa A): %+v", len(lido), lido)
	}
	in := lido[0]
	if in.ItemCode != item || len(in.Buckets) != 2 || !in.Stock.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("perfil/estoque lido errado: %+v", in)
	}
	// A sugestão firmada não conta como planejada pendente.
	if in.PlannedOrders != 1 || in.FirstPlannedStart == nil {
		t.Fatalf("planejadas pendentes: %d (%v)", in.PlannedOrders, in.FirstPlannedStart)
	}
	if outra, err := repo.Coverage(b.Ctx, mrp_report_uc.Filter{PlanCode: &plano}); err != nil || len(outra) != 1 || outra[0].ItemCode != itemB || outra[0].PlannedOrders != 0 {
		t.Fatalf("empresa B vê as planejadas da A: %+v (%v)", outra, err)
	}
}
