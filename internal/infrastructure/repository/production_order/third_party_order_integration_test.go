//go:build integration

package production_order_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func TestManualProductionOrderCreatesExternalServiceOrderAtomically(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(base, `SELECT MIN(id) FROM enterprise`).Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	itemCode, operationCode, routeCode, orderNumber := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	actor := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO items(code,warehouse_code,warehouse_unit_of_measurement,created_by) VALUES($1,$1,'UN',$2)`, itemCode, actor); err != nil {
		t.Fatal(err)
	}
	var operationID, routeID int64
	if err := pool.QueryRow(ctx, `INSERT INTO operations(code,name,origin,created_by,lead_time_days,third_party_remittance) VALUES($1,'External manual OF','TERCEIROS',$2,3,'ORDER_ITEM') RETURNING id`, operationCode, actor).Scan(&operationID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO manufacturing_routes(code,item_code,alternative,is_standard,is_active,created_by) VALUES($1,$2,1,TRUE,TRUE,$3) RETURNING id`, routeCode, itemCode, actor).Scan(&routeID); err != nil {
		t.Fatal(err)
	}
	var routeOperationID int64
	if err := pool.QueryRow(ctx, `INSERT INTO route_operations(route_id,sequence,operation_id,is_active) VALUES($1,10,$2,TRUE) RETURNING id`, routeID, operationID).Scan(&routeOperationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(base, `DELETE FROM production_orders WHERE order_number=$1 AND enterprise_id=$2`, orderNumber, enterpriseID)
		_, _ = pool.Exec(base, `DELETE FROM manufacturing_routes WHERE id=$1`, routeID)
		_, _ = pool.Exec(base, `DELETE FROM operations WHERE id=$1`, operationID)
		_, _ = pool.Exec(base, `DELETE FROM items WHERE code=$1`, itemCode)
	})
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	created, err := production_order.NewProductionOrderRepositoryPGX(pool).Create(ctx, &entity.ProductionOrder{
		OrderNumber: orderNumber, ItemCode: itemCode, PlannedQty: 12, Status: entity.StatusOpen,
		StartDate: &start, IsActive: true, CreatedBy: actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	var serviceCount int
	var gotRouteOperationID int64
	var status, remittance string
	var dueDate time.Time
	if err = pool.QueryRow(ctx, `SELECT COUNT(*),MIN(route_operation_id),MIN(status),MIN(remittance_type),MIN(due_date) FROM third_party_service_orders WHERE enterprise_id=$1 AND production_order_id=$2`, enterpriseID, created.ID).Scan(&serviceCount, &gotRouteOperationID, &status, &remittance, &dueDate); err != nil {
		t.Fatal(err)
	}
	if serviceCount != 1 || gotRouteOperationID != routeOperationID || status != "FIRM" || remittance != "ORDER_ITEM" || !dueDate.Equal(start.AddDate(0, 0, 3)) {
		t.Fatalf("service order count=%d route_op=%d status=%s remittance=%s due=%s", serviceCount, gotRouteOperationID, status, remittance, dueDate)
	}
}
