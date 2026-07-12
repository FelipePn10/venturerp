//go:build integration

package production_order_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	productionentity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func TestRegisterDeliveryWithMovements_RollsBackEverythingWhenStockFails(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(base, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	uid := uuid.New()
	itemCode := testutil.UniqueCode()
	testutil.Exec(t, pool, "INSERT INTO items (code, warehouse_code, created_by) VALUES ($1,$2,$3)", itemCode, itemCode, uid)
	defer testutil.Exec(t, pool, "DELETE FROM items WHERE code=$1", itemCode)

	var orderID int64
	if err := pool.QueryRow(ctx, `INSERT INTO production_orders
		(order_number,item_code,planned_qty,produced_qty,status,created_by,enterprise_id)
		VALUES ($1,$2,10,0,'RELEASED',$3,$4) RETURNING id`,
		testutil.UniqueCode(), itemCode, uid, enterpriseID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM production_orders WHERE id=$1", orderID)

	delivery := &productionentity.ProductionDelivery{ProductionOrderID: orderID, Quantity: decimal.NewFromInt(2),
		IdempotencyKey: uuid.NewString(), MovementClass: "EPP", WarehouseID: itemCode, CreatedBy: uid,
		Lines: []productionentity.ProductionDeliveryLine{{MovementClass: "EPP", Quantity: decimal.NewFromInt(2)}}}
	refType, refCode := stockentity.ReferenceTypeProductionOrder, orderID
	movements := []*stockentity.StockMovement{{ItemCode: itemCode, WarehouseID: itemCode,
		MovementType: "EPP", Quantity: 1e20, ReferenceType: &refType, ReferenceCode: &refCode, CreatedBy: uid}}

	if _, err := productionrepo.NewProductionOrderRepositoryPGX(pool).RegisterDeliveryWithMovements(ctx, delivery, movements); err == nil {
		t.Fatal("expected stock numeric overflow")
	}
	var deliveries int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM production_deliveries WHERE production_order_id=$1", orderID).Scan(&deliveries); err != nil {
		t.Fatal(err)
	}
	var produced float64
	if err := pool.QueryRow(ctx, "SELECT produced_qty FROM production_orders WHERE id=$1", orderID).Scan(&produced); err != nil {
		t.Fatal(err)
	}
	if deliveries != 0 || produced != 0 {
		t.Fatalf("transaction leaked partial state: deliveries=%d produced=%v", deliveries, produced)
	}
}
