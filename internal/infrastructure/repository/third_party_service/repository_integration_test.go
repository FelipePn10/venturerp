//go:build integration

package third_party_service_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	domain "github.com/FelipePn10/panossoerp/internal/domain/third_party_service"
	repoimpl "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/third_party_service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestPriceLifecycleResolutionAndTenantIsolation(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	var enterpriseID int64
	if e := pool.QueryRow(base, `SELECT MIN(id) FROM enterprise`).Scan(&enterpriseID); e != nil {
		t.Fatal(e)
	}
	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	uid := uuid.New()
	suffix := time.Now().UnixNano()
	supplierCode := suffix%1000000000 + 7000000000
	operationCode := supplierCode + 1
	var supplierID, operationID int64
	if e := pool.QueryRow(base, `INSERT INTO suppliers(code,name,document_type,document_number,created_by) VALUES($1,'Third party test','ESTRANGEIRO',$2,$3) RETURNING id`, supplierCode, fmt.Sprintf("T%d", suffix), uid).Scan(&supplierID); e != nil {
		t.Fatal(e)
	}
	if e := pool.QueryRow(base, `INSERT INTO operations(code,name,origin,created_by) VALUES($1,'External test','TERCEIROS',$2) RETURNING id`, operationCode, uid).Scan(&operationID); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		pool.Exec(base, `DELETE FROM operations WHERE id=$1`, operationID)
		pool.Exec(base, `DELETE FROM suppliers WHERE id=$1`, supplierID)
	})
	repo := repoimpl.New(pool)
	var itemCode int64
	var e error
	if e = pool.QueryRow(base, `SELECT MIN(code) FROM items`).Scan(&itemCode); e != nil || itemCode == 0 {
		t.Fatalf("test item: %v", e)
	}
	var oldAttributes []byte
	if e = pool.QueryRow(base, `SELECT pdm_attributes FROM items WHERE code=$1`, itemCode).Scan(&oldAttributes); e != nil {
		t.Fatal(e)
	}
	if _, e = pool.Exec(base, `UPDATE items SET pdm_attributes='[{"Name":"BASE","Value":"6.25"},{"Name":"COLOR","Value":"BLUE"}]'::jsonb WHERE code=$1`, itemCode); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { pool.Exec(base, `UPDATE items SET pdm_attributes=$2 WHERE code=$1`, itemCode, oldAttributes) })
	answer := "BLUE"
	ref := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	p := &domain.Price{ItemCode: itemCode, Mask: "", SupplierCode: supplierCode, OperationID: operationID, UOM: "UN", ReferenceDate: ref, Preferred: true, UnitPrice: decimal.RequireFromString("10.50"), Formula: "BASE*2", FreightType: "PERCENT", FreightValue: decimal.NewFromInt(10), TaxPercent: decimal.NewFromInt(5), CreatedBy: uid, Rules: []domain.PriceRule{{Characteristic: "COLOR", Answer: &answer}}}
	created, e := repo.CreatePrice(ctx, p, "initial contract")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		pool.Exec(base, `DELETE FROM third_party_service_price_history WHERE enterprise_id=$1 AND price_id IN (SELECT id FROM third_party_service_prices WHERE enterprise_id=$1 AND item_code=$2)`, enterpriseID, p.ItemCode)
		pool.Exec(base, `DELETE FROM third_party_service_prices WHERE enterprise_id=$1 AND item_code=$2`, enterpriseID, p.ItemCode)
	})
	resolved, e := repo.ResolvePrice(ctx, p.ItemCode, "", supplierCode, operationID, ref.AddDate(0, 1, 0), map[string]string{"COLOR": "BLUE"})
	if e != nil || resolved.ID != created.ID || !resolved.UnitPrice.Equal(decimal.RequireFromString("12.5")) {
		t.Fatalf("resolved=%+v err=%v", resolved, e)
	}
	if _, e = repo.ResolvePrice(ctx, p.ItemCode, "", supplierCode, operationID, ref.AddDate(0, 1, 0), map[string]string{"COLOR": "RED"}); e == nil {
		t.Fatal("rule mismatch must not resolve")
	}
	history, e := repo.History(ctx, created.ID)
	if e != nil || len(history) != 1 || history[0].Action != "CREATE" {
		t.Fatalf("history=%+v err=%v", history, e)
	}
	if _, e = repo.Readjust(ctx, []int64{created.ID, 9223372036854775807}, decimal.NewFromInt(10), ref.AddDate(0, 1, 0), "annual adjustment", uid); e == nil {
		t.Fatal("batch containing an unknown price must fail")
	}
	var rolledBack int
	if e = pool.QueryRow(base, `SELECT COUNT(*) FROM third_party_service_prices WHERE enterprise_id=$1 AND item_code=$2 AND reference_date=$3`, enterpriseID, p.ItemCode, ref.AddDate(0, 1, 0)).Scan(&rolledBack); e != nil || rolledBack != 0 {
		t.Fatalf("readjustment was not atomic: count=%d err=%v", rolledBack, e)
	}
	adjusted, e := repo.Readjust(ctx, []int64{created.ID}, decimal.NewFromInt(10), ref.AddDate(0, 1, 0), "annual adjustment", uid)
	if e != nil || len(adjusted) != 1 || !adjusted[0].UnitPrice.Equal(decimal.RequireFromString("11.55")) {
		t.Fatalf("adjusted=%+v err=%v", adjusted, e)
	}
	routeCode := operationCode + 100
	var routeID, routeOpID, productionID int64
	if e = pool.QueryRow(base, `INSERT INTO manufacturing_routes(code,item_code,alternative,is_standard,created_by) VALUES($1,$2,32760,FALSE,$3) RETURNING id`, routeCode, itemCode, uid).Scan(&routeID); e != nil {
		t.Fatal(e)
	}
	if e = pool.QueryRow(base, `INSERT INTO route_operations(route_id,sequence,operation_id) VALUES($1,32760,$2) RETURNING id`, routeID, operationID).Scan(&routeOpID); e != nil {
		t.Fatal(e)
	}
	if _, e = pool.Exec(base, `UPDATE operations SET supplier_id=$2,service_item_code=$3,lead_time_days=5 WHERE id=$1`, operationID, supplierID, itemCode); e != nil {
		t.Fatal(e)
	}
	if e = pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,mask,planned_qty,status,start_date,end_date,route_id,enterprise_id,created_by) VALUES($1,$2,'',10,'OPEN','2026-07-13','2026-07-20',$3,$4,$5) RETURNING id`, routeCode, itemCode, routeID, enterpriseID, uid).Scan(&productionID); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		pool.Exec(base, `DELETE FROM production_orders WHERE id=$1`, productionID)
		pool.Exec(base, `DELETE FROM manufacturing_routes WHERE id=$1`, routeID)
	})
	orders, e := repo.CreateOrdersForProduction(ctx, productionID, uid)
	if e != nil || len(orders) != 1 || orders[0].RouteOperationID != routeOpID || orders[0].SupplierCode == nil || *orders[0].SupplierCode != supplierCode {
		t.Fatalf("orders=%+v err=%v", orders, e)
	}
	planCode := suffix%1000000000 + 5000000000
	var suggestionCode int64
	if e = pool.QueryRow(base, `INSERT INTO mrp_planned_suggestions(plan_code,item_code,mask,quantity,need_date,start_date,order_type,demand_type,llc,enterprise_id,order_number,route_operation_id,operation_id,supplier_code,service_item_code,remittance_type,notes)
		VALUES($1,$2,'BLUE',10,'2026-07-25','2026-07-20','SERVICO','EXTERNA',0,$3,$4,$5,$6,$7,$2,'DEMAND_ITEMS','planned external operation') RETURNING code`, planCode, itemCode, enterpriseID, routeCode+500, routeOpID, operationID, supplierCode).Scan(&suggestionCode); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _, _ = pool.Exec(base, `DELETE FROM mrp_planned_suggestions WHERE code=$1`, suggestionCode) })
	plannedOrders, e := repo.ListOrders(ctx, domain.OrderFilter{PlanCode: &planCode, Statuses: []string{"PLANNED"}})
	if e != nil || len(plannedOrders) != 1 || plannedOrders[0].PlannedSuggestionCode == nil || *plannedOrders[0].PlannedSuggestionCode != suggestionCode || plannedOrders[0].OperationID != operationID {
		t.Fatalf("planned service orders=%+v err=%v", plannedOrders, e)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, createErr := repo.CreateOrdersForProduction(ctx, productionID, uid)
			errs <- createErr
		}()
	}
	wg.Wait()
	close(errs)
	for createErr := range errs {
		if createErr != nil {
			t.Fatalf("concurrent idempotent creation: %v", createErr)
		}
	}
	var orderCount int
	if e = pool.QueryRow(base, `SELECT COUNT(*) FROM third_party_service_orders WHERE enterprise_id=$1 AND production_order_id=$2`, enterpriseID, productionID).Scan(&orderCount); e != nil || orderCount != 1 {
		t.Fatalf("concurrent creation count=%d err=%v", orderCount, e)
	}
	if e = repo.LinkRequisitionToProduction(ctx, productionID, 88001); e != nil {
		t.Fatal(e)
	}
	linkedOrder, e := repo.GetOrder(ctx, orders[0].ID)
	if e != nil || linkedOrder.PurchaseRequisitionCode == nil || *linkedOrder.PurchaseRequisitionCode != 88001 {
		t.Fatalf("requisition not linked: order=%+v err=%v", linkedOrder, e)
	}
	if _, e = repo.UpdateOrderStatus(ctx, orders[0].ID, "RELEASED_WITH_PO", nil, nil, uid); e == nil {
		t.Fatal("release with purchase order must require its code")
	}
	poCode := int64(991)
	if _, e = repo.UpdateOrderStatus(ctx, orders[0].ID, "RELEASED_WITH_PO", nil, &poCode, uid); e != nil {
		t.Fatal(e)
	}
	if _, e = repo.AddMovement(ctx, orders[0].ID, domain.Movement{MovementType: "RETURN", Quantity: decimal.NewFromInt(1), OccurredAt: time.Now(), IdempotencyKey: fmt.Sprintf("invalid-return-%d", suffix), CreatedBy: uid}); e == nil {
		t.Fatal("return without remittance must fail")
	}
	if _, e = repo.AddMovement(ctx, orders[0].ID, domain.Movement{MovementType: "REMITTANCE", Quantity: decimal.NewFromInt(4), OccurredAt: time.Now(), IdempotencyKey: fmt.Sprintf("remittance-%d", suffix), CreatedBy: uid}); e != nil {
		t.Fatal(e)
	}
	if _, e = repo.AddMovement(ctx, orders[0].ID, domain.Movement{MovementType: "RETURN", Quantity: decimal.NewFromInt(4), OccurredAt: time.Now(), IdempotencyKey: fmt.Sprintf("return-%d", suffix), CreatedBy: uid}); e != nil {
		t.Fatal(e)
	}
	returnedOrder, e := repo.GetOrder(ctx, orders[0].ID)
	if e != nil || !returnedOrder.FulfilledQuantity.IsZero() {
		t.Fatalf("logistical return must not duplicate receipt: order=%+v err=%v", returnedOrder, e)
	}
	movement, e := repo.AddMovement(ctx, orders[0].ID, domain.Movement{MovementType: "RECEIPT", Quantity: decimal.NewFromInt(4), OccurredAt: time.Now(), IdempotencyKey: fmt.Sprintf("receipt-%d", suffix), CreatedBy: uid})
	if e != nil || movement.ID == 0 {
		t.Fatalf("movement=%+v err=%v", movement, e)
	}
	repeated, e := repo.AddMovement(ctx, orders[0].ID, domain.Movement{MovementType: "RECEIPT", Quantity: decimal.NewFromInt(4), OccurredAt: time.Now(), IdempotencyKey: fmt.Sprintf("receipt-%d", suffix), CreatedBy: uid})
	if e != nil || repeated.ID != movement.ID {
		t.Fatalf("idempotent movement=%+v err=%v", repeated, e)
	}
	gotOrder, e := repo.GetOrder(ctx, orders[0].ID)
	if e != nil || !gotOrder.FulfilledQuantity.Equal(decimal.NewFromInt(4)) {
		t.Fatalf("order=%+v err=%v", gotOrder, e)
	}
	orderHistory, e := repo.OrderHistory(ctx, orders[0].ID)
	if e != nil || len(orderHistory) < 3 {
		t.Fatalf("order history=%+v err=%v", orderHistory, e)
	}
	global, e := repo.UpsertGlobalConversion(ctx, domain.GlobalConversion{FromUOM: "CX", ToUOM: "UN", Factor: decimal.NewFromInt(12), CreatedBy: uid})
	if e != nil || global.ID == 0 {
		t.Fatalf("global conversion=%+v err=%v", global, e)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(base, `DELETE FROM global_unit_conversions WHERE id=$1`, global.ID)
	})
	globals, e := repo.ListGlobalConversions(ctx)
	if e != nil || len(globals) == 0 {
		t.Fatalf("global conversions=%+v err=%v", globals, e)
	}
	var otherCode int64
	if e = pool.QueryRow(base, `SELECT COALESCE(MAX(code),0)+1 FROM enterprise`).Scan(&otherCode); e != nil {
		t.Fatal(e)
	}
	var otherID int64
	if e = pool.QueryRow(base, `INSERT INTO enterprise(code,name) VALUES($1,'Other third party tenant') RETURNING id`, otherCode).Scan(&otherID); e != nil {
		t.Fatal(e)
	}
	defer pool.Exec(base, `DELETE FROM enterprise WHERE id=$1`, otherID)
	otherCtx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: otherID})
	if _, e = repo.GetPrice(otherCtx, created.ID); e == nil {
		t.Fatal("price leaked across tenants")
	}
}
