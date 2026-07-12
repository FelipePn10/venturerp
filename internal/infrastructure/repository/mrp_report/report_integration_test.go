//go:build integration

package mrp_report_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_report_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_report"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func TestAllMRPReportsExecuteWithTenantScope(t *testing.T) {
	_, pool := testutil.Queries(t)
	ctx := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(ctx, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	repo := mrp_report.New(pool)
	filter := mrp_report_uc.Filter{}
	checks := []struct {
		name string
		run  func() error
	}{
		{"profile", func() error { _, err := repo.Profile(ctx, filter); return err }},
		{"availability", func() error { _, err := repo.Availability(ctx, filter); return err }},
		{"grouped needs", func() error { _, err := repo.GroupedNeeds(ctx, filter); return err }},
		{"explosion", func() error {
			_, err := repo.Explosion(ctx, testutil.UniqueCode(), decimal.NewFromInt(1), nil, filter)
			return err
		}},
		{"reorder point", func() error { _, err := repo.ReorderPoint(ctx, filter); return err }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.run(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAvailabilityExplodesManualQuantityAndAppliesLayout(t *testing.T) {
	_, pool := testutil.Queries(t)
	ctx := context.Background()
	var enterpriseID int64
	var userID string
	if err := pool.QueryRow(ctx, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT created_by::text FROM items LIMIT 1").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	parent, child := testutil.UniqueCode(), testutil.UniqueCode()
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM item_structures WHERE parent_code=$1", parent)
		testutil.Exec(t, pool, "DELETE FROM items WHERE code=ANY($1::bigint[])", []int64{parent, child})
	})
	testutil.Exec(t, pool, "INSERT INTO items (code,warehouse_code,created_by) VALUES ($1,$1,$3),($2,$2,$3)", parent, child, userID)
	testutil.Exec(t, pool, "INSERT INTO item_structures (parent_code,child_code,quantity,sequence,created_by) VALUES ($1,$2,2,1,$3)", parent, child, userID)
	repo := mrp_report.New(pool)
	rows, err := repo.Availability(ctx, mrp_report_uc.Filter{ItemCode: &parent, Quantity: decimal.NewFromInt(3), Layout: "AMBOS"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%d want 2: %+v", len(rows), rows)
	}
	if rows[0].RowType != "ITEM_PEDIDO" || !rows[0].Demand.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("root=%+v", rows[0])
	}
	if rows[1].RowType != "NECESSIDADE" || rows[1].ItemCode != child || !rows[1].Demand.Equal(decimal.NewFromInt(6)) {
		t.Fatalf("child=%+v", rows[1])
	}
	needs, err := repo.Availability(ctx, mrp_report_uc.Filter{ItemCode: &parent, Quantity: decimal.NewFromInt(3), Layout: "NECESSIDADES"})
	if err != nil {
		t.Fatal(err)
	}
	if len(needs) != 1 || needs[0].ItemCode != child {
		t.Fatalf("needs=%+v", needs)
	}
}

func TestProfileReturnsPersistedOriginsAndTenantDrawings(t *testing.T) {
	_, pool := testutil.Queries(t)
	ctx := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(ctx, "SELECT id FROM enterprise ORDER BY id LIMIT 1").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	var userID string
	if err := pool.QueryRow(ctx, "SELECT created_by::text FROM items LIMIT 1").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	item, plan := testutil.UniqueCode(), testutil.UniqueCode()
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM drawings WHERE item_code=$1", item)
		testutil.Exec(t, pool, "DELETE FROM mrp_profile_details WHERE plan_code=$1", plan)
		testutil.Exec(t, pool, "DELETE FROM mrp_item_profiles WHERE plan_code=$1", plan)
		testutil.Exec(t, pool, "DELETE FROM production_plans WHERE code=$1", plan)
		testutil.Exec(t, pool, "DELETE FROM items WHERE code=$1", item)
	})
	testutil.Exec(t, pool, "INSERT INTO items(code,warehouse_code,created_by) VALUES($1,$1,$2)", item, userID)
	testutil.Exec(t, pool, "INSERT INTO production_plans(code,name,created_by,enterprise_id) VALUES($1,'report',$2,$3)", plan, userID, enterpriseID)
	testutil.Exec(t, pool, "INSERT INTO mrp_item_profiles(item_code,plan_code,calculation_date,demand,orders_planned,orders_firm,stock_projected,llc,need_date,enterprise_id) VALUES($1,$2,CURRENT_DATE,4,2,0,-2,0,CURRENT_DATE,$3)", item, plan, enterpriseID)
	testutil.Exec(t, pool, "INSERT INTO mrp_profile_details(enterprise_id,plan_code,item_code,need_date,detail_type,source_code,quantity) VALUES($1,$2,$3,CURRENT_DATE,'SALES_ORDER',123,4)", enterpriseID, plan, item)
	for index, enterprise := range []*int64{&enterpriseID, nil} {
		var drawingID int64
		code := "DRAW-TENANT-" + string(rune('A'+index))
		if err := pool.QueryRow(ctx, "INSERT INTO drawings(code,item_code,created_by,enterprise_id) VALUES($1,$2,$3,$4) RETURNING id", code, item, userID, enterprise).Scan(&drawingID); err != nil {
			t.Fatal(err)
		}
		testutil.Exec(t, pool, "INSERT INTO drawing_revisions(drawing_id,revision,is_current) VALUES($1,'R1',TRUE)", drawingID)
	}
	repo := mrp_report.New(pool)
	result, err := repo.Profile(ctx, mrp_report_uc.Filter{PlanCode: &plan, Layout: "ANALITICO", IncludeDrawings: true})
	if err != nil {
		t.Fatal(err)
	}
	var summary, detail *mrp_report_uc.ReportRow
	for i := range result {
		if result[i].RowType == "DETALHE" {
			detail = &result[i]
		} else {
			summary = &result[i]
		}
	}
	if detail == nil || detail.SourceType != "SALES_ORDER" || detail.SourceCode == nil || *detail.SourceCode != 123 {
		t.Fatalf("detail=%+v", detail)
	}
	if summary == nil || len(summary.DrawingCodes) != 1 || summary.DrawingCodes[0] != "DRAW-TENANT-AR1" {
		t.Fatalf("summary=%+v", summary)
	}
}

func TestReorderPointTraversesReleasedAndBlockedSalesOrderStructures(t *testing.T) {
	_, pool := testutil.Queries(t)
	ctx := context.Background()
	var enterpriseID, enterpriseCode int64
	var userID string
	if err := pool.QueryRow(ctx, "SELECT id,code FROM enterprise ORDER BY id LIMIT 1").Scan(&enterpriseID, &enterpriseCode); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT created_by::text FROM items LIMIT 1").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	parent, child := testutil.UniqueCode(), testutil.UniqueCode()
	order1, order2 := testutil.UniqueCode(), testutil.UniqueCode()
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM sales_order_items WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE order_number=ANY($1::bigint[]) AND enterprise_code=$2)", []int64{order1, order2}, enterpriseCode)
		testutil.Exec(t, pool, "DELETE FROM sales_orders WHERE order_number=ANY($1::bigint[]) AND enterprise_code=$2", []int64{order1, order2}, enterpriseCode)
		testutil.Exec(t, pool, "DELETE FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2", enterpriseID, child)
		testutil.Exec(t, pool, "DELETE FROM item_structures WHERE parent_code=$1", parent)
		testutil.Exec(t, pool, "DELETE FROM items WHERE code=ANY($1::bigint[])", []int64{parent, child})
	})
	testutil.Exec(t, pool, "INSERT INTO items(code,warehouse_code,created_by) VALUES($1,$1,$3),($2,$2,$3)", parent, child, userID)
	testutil.Exec(t, pool, "INSERT INTO item_structures(parent_code,child_code,quantity,sequence,created_by) VALUES($1,$2,2,1,$3)", parent, child, userID)
	testutil.Exec(t, pool, "INSERT INTO stock_balances(item_code,warehouse_id,quantity,enterprise_id) VALUES($1,1,0,$2)", child, enterpriseID)
	for _, order := range []struct {
		number, qty int64
		blocked     bool
	}{{order1, 2, false}, {order2, 3, true}} {
		var code int64
		if err := pool.QueryRow(ctx, "INSERT INTO sales_orders(order_number,enterprise_code,is_blocked,created_by) VALUES($1,$2,$3,$4) RETURNING code", order.number, enterpriseCode, order.blocked, userID).Scan(&code); err != nil {
			t.Fatal(err)
		}
		testutil.Exec(t, pool, "INSERT INTO sales_order_items(sales_order_code,item_code,requested_qty,status) VALUES($1,$2,$3,'OPEN')", code, parent, order.qty)
	}
	repo := mrp_report.New(pool)
	released, err := repo.ReorderPoint(ctx, mrp_report_uc.Filter{ItemCode: &child, OrderPosition: "LIBERADOS"})
	if err != nil {
		t.Fatal(err)
	}
	all, err := repo.ReorderPoint(ctx, mrp_report_uc.Filter{ItemCode: &child, OrderPosition: "LIBERADOS_E_BLOQUEADOS"})
	if err != nil {
		t.Fatal(err)
	}
	if len(released) != 1 || !released[0].Demand.Equal(decimal.NewFromInt(4)) {
		t.Fatalf("released=%+v", released)
	}
	if len(all) != 1 || !all[0].Demand.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("all=%+v", all)
	}
}
