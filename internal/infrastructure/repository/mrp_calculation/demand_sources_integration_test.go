//go:build integration

package mrp_calculation

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func TestDemandSourcesAreTenantFilteredAndResolveClassificationDescendants(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	userID := uuid.New()
	const enterpriseCode, otherCode, orderNumber, itemCode, otherItemCode, maskCode, divisionCode, warehouseCode, planCode = 920001, 920002, 929001, 921001, 921002, 929901, 928001, 927001, 929101
	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM item_classification_assignments WHERE item_code IN ($1,$2)`, itemCode, otherItemCode)
		_, _ = pool.Exec(ctx, `DELETE FROM item_classifications WHERE mask_id IN (SELECT id FROM item_classification_masks WHERE code=$1)`, maskCode)
		_, _ = pool.Exec(ctx, `DELETE FROM item_classification_masks WHERE code=$1`, maskCode)
		_, _ = pool.Exec(ctx, `DELETE FROM sales_order_items WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE order_number=$1 AND enterprise_code IN ($2,$3))`, orderNumber, enterpriseCode, otherCode)
		_, _ = pool.Exec(ctx, `DELETE FROM sales_orders WHERE order_number=$1 AND enterprise_code IN ($2,$3)`, orderNumber, enterpriseCode, otherCode)
		_, _ = pool.Exec(ctx, `DELETE FROM sales_divisions WHERE code=$1`, divisionCode)
		_, _ = pool.Exec(ctx, `DELETE FROM production_plans WHERE code=$1`, planCode)
		_, _ = pool.Exec(ctx, `DELETE FROM enterprise WHERE code IN ($1,$2)`, enterpriseCode, otherCode)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email='mrp-demand-920001@test.local'`)
	}
	cleanup()
	testutil.Exec(t, pool, `INSERT INTO users (id,name,email,password) VALUES ($1,'MRP demand','mrp-demand-920001@test.local','x')`, userID)
	testutil.Exec(t, pool, `INSERT INTO enterprise (code,name,created_by) VALUES ($1,'Demand owner',$3),($2,'Other demand owner',$3)`, enterpriseCode, otherCode, userID)
	var enterpriseID, otherID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM enterprise WHERE code=$1`, enterpriseCode).Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM enterprise WHERE code=$1`, otherCode).Scan(&otherID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO production_plans (code,name,created_by,enterprise_id) VALUES ($1,'Demand plan',$2,$3)`, planCode, userID, enterpriseID)
	var divisionID int64
	if err := pool.QueryRow(ctx, `INSERT INTO sales_divisions (code,description,is_technical_assistance,created_by,enterprise_id) VALUES ($1,'Technical assistance',TRUE,$2,$3) RETURNING id`, divisionCode, userID, enterpriseID).Scan(&divisionID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO sales_orders (order_number,enterprise_code,status,origin,delivery_date,sales_division_code,created_by) VALUES ($1,$2,'R','NORMAL','2026-08-10',$4,$5),($1,$3,'R','INTER_FACTORY','2026-08-10',NULL,$5)`, orderNumber, enterpriseCode, otherCode, divisionID, userID)
	var orderCode, otherOrderCode int64
	if err := pool.QueryRow(ctx, `SELECT code FROM sales_orders WHERE order_number=$1 AND enterprise_code=$2`, orderNumber, enterpriseCode).Scan(&orderCode); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT code FROM sales_orders WHERE order_number=$1 AND enterprise_code=$2`, orderNumber, otherCode).Scan(&otherOrderCode); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO sales_order_items (sales_order_code,item_code,warehouse_code,requested_qty,attended_qty,cancelled_qty) VALUES ($1,$2,$5,10,3,2),($3,$4,NULL,99,0,0)`, orderCode, itemCode, otherOrderCode, otherItemCode, warehouseCode)
	testutil.Exec(t, pool, `INSERT INTO item_classification_masks (code,mask,description) VALUES ($1,'99.999.999','MRP test')`, maskCode)
	var maskID, parentID, childID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM item_classification_masks WHERE code=$1`, maskCode).Scan(&maskID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO item_classifications (code,mask_id,level,description) VALUES ('10',$1,1,'Parent')`, maskID)
	if err := pool.QueryRow(ctx, `SELECT id FROM item_classifications WHERE mask_id=$1 AND code='10'`, maskID).Scan(&parentID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO item_classifications (code,mask_id,parent_id,level,description) VALUES ('10.100',$1,$2,2,'Child')`, maskID, parentID)
	if err := pool.QueryRow(ctx, `SELECT id FROM item_classifications WHERE mask_id=$1 AND code='10.100'`, maskID).Scan(&childID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO item_classification_assignments (enterprise_id,item_code,classification_id) VALUES ($1,$2,$3),($4,$5,$3)`, enterpriseID, itemCode, childID, otherID, otherItemCode)
	t.Cleanup(func() {
		cleanup()
	})

	repo := NewMRPCalculationRepositorySQLC(sqlc.New(pool), pool)
	tenantCtx := context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	demands, err := repo.ListOpenSalesOrderDemands(tenantCtx, planCode, nil)
	if err != nil || len(demands) != 1 || demands[0].ItemCode != itemCode || demands[0].Quantity != 5 {
		t.Fatalf("unexpected tenant sales demands: %#v, %v", demands, err)
	}
	if !demands[0].TechnicalAssistance || demands[0].WarehouseCode == nil || *demands[0].WarehouseCode != warehouseCode {
		t.Fatalf("technical-assistance source metadata was not propagated: %#v", demands[0])
	}
	testutil.Exec(t, pool, `INSERT INTO production_plan_inter_factories (plan_code,enterprise_id,source_enterprise_id,auto_release) VALUES ($1,$2,$3,TRUE)`, planCode, enterpriseID, otherID)
	demands, err = repo.ListOpenSalesOrderDemands(tenantCtx, planCode, nil)
	if err != nil || len(demands) != 2 {
		t.Fatalf("configured inter-factory demand was not loaded: %#v, %v", demands, err)
	}
	var interFactoryDemandFound bool
	for _, demand := range demands {
		if demand.InterFactory {
			interFactoryDemandFound = demand.SourceEnterpriseCode != nil && *demand.SourceEnterpriseCode == otherCode && demand.AutoRelease && demand.DemandType == "SALES_ORDER"
		}
	}
	if !interFactoryDemandFound {
		t.Fatalf("inter-factory metadata was not propagated: %#v", demands)
	}
	items, err := repo.ResolveClassificationItemCodes(tenantCtx, "929901", []string{"10"})
	if err != nil || len(items) != 1 || items[0] != itemCode {
		t.Fatalf("unexpected classified items: %#v, %v", items, err)
	}
}
