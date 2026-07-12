//go:build integration

package production_order_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestPhase4MaintenanceLotPoliciesAndScrapControls(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(base, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	repo := production_order.NewProductionOrderRepositoryPGX(pool)
	uid := uuid.New()
	orderItem, component, scrapItem := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	var orderID, kanbanID int64
	if err := pool.QueryRow(ctx, `INSERT INTO production_orders(order_number,item_code,planned_qty,scrapped_qty,status,origin_type,created_by,enterprise_id) VALUES($1,$2,5,2,'RELEASED','MANUAL',$3,$4) RETURNING id`, testutil.UniqueCode(), orderItem, uid, enterpriseID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,origin_type,created_by,enterprise_id) VALUES($1,$2,5,'RELEASED','KANBAN',$3,$4) RETURNING id`, testutil.UniqueCode(), orderItem, uid, enterpriseID).Scan(&kanbanID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM item_unit_conversions WHERE item_code=$1", scrapItem)
		pool.Exec(ctx, "DELETE FROM stock_movements WHERE enterprise_id=$1 AND reference_type LIKE 'PRODUCTION_SCRAP%'", enterpriseID)
		pool.Exec(ctx, "DELETE FROM stock_balances WHERE enterprise_id=$1 AND item_code=ANY($2)", enterpriseID, []int64{component, scrapItem})
		pool.Exec(ctx, "DELETE FROM manufacturing_stock_closed_periods WHERE enterprise_id=$1", enterpriseID)
		pool.Exec(ctx, "DELETE FROM manufacturing_stock_item_controls WHERE enterprise_id=$1 AND item_code=ANY($2)", enterpriseID, []int64{component, scrapItem})
		pool.Exec(ctx, "DELETE FROM manufacturing_stock_parameters WHERE enterprise_id=$1", enterpriseID)
		pool.Exec(ctx, "DELETE FROM production_orders WHERE id=ANY($1)", []int64{orderID, kanbanID})
	})
	views, err := repo.GetMaintenance(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range views {
		if v.ProductionOrder.ID == kanbanID {
			t.Fatal("general maintenance must hide Kanban order")
		}
	}
	if _, err := repo.GetMaintenance(ctx, &kanbanID); err == nil || !strings.Contains(err.Error(), "Kanban") {
		t.Fatalf("specific Kanban maintenance err=%v", err)
	}
	tomorrow := time.Now().AddDate(0, 0, 1)
	lot, err := repo.ConfigureTemporaryLot(ctx, entity.TemporaryProductionLot{ProductionOrderID: orderID, Lot: "TEMP-OF", ManufacturedOn: time.Now(), ExpiresOn: tomorrow})
	if err != nil || lot.Lot != "TEMP-OF" {
		t.Fatalf("temporary lot=%+v err=%v", lot, err)
	}
	material, err := repo.AddMaterial(ctx, &entity.ProductionOrderMaterial{ProductionOrderID: orderID, Kind: entity.MaterialDemand, ItemCode: component, Quantity: decimal.NewFromInt(3), WarehouseID: 1, CreatedBy: uid})
	if err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, "UPDATE production_order_materials SET attended_quantity=3 WHERE id=$1", material.ID)
	if err := repo.ConfigureManufacturingStock(ctx, entity.ManufacturingStockParameters{LotReturnMode: "E", AutoIssueLots: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AllocateLotsWithPolicy(ctx, material.ID, "RETURN", []entity.LotAllocation{{WarehouseID: 1, Lot: "NEW", Quantity: decimal.NewFromInt(1)}}, true, uid); err == nil {
		t.Fatal("mode E must reject a lot not used by requisition")
	}
	if err := repo.ConfigureManufacturingStock(ctx, entity.ManufacturingStockParameters{LotReturnMode: "A", AutoIssueLots: true}); err != nil {
		t.Fatal(err)
	}
	automaticReturn, err := repo.AllocateLotsWithPolicy(ctx, material.ID, "RETURN", nil, false, uid)
	if err != nil || len(automaticReturn) != 1 || !strings.HasPrefix(automaticReturn[0].Lot, "OF-") {
		t.Fatalf("parameter 44 mode A allocation=%+v err=%v", automaticReturn, err)
	}
	if err := repo.ConfigureManufacturingItemStock(ctx, entity.ManufacturingItemStockControl{ItemCode: scrapItem, StockUOM: "KG", ControlsLot: true, ControlsAddress: true, InventoryGroupType: "SECONDARY_MATERIAL"}); err == nil {
		t.Fatal("configuration must reject nonexistent item")
	}
	testutil.Exec(t, pool, `INSERT INTO manufacturing_stock_item_controls(enterprise_id,item_code,stock_uom,controls_lot,controls_address,inventory_group_type) VALUES($1,$2,'KG',true,true,'SECONDARY_MATERIAL'),($1,$3,'UN',false,false,'STANDARD')`, enterpriseID, scrapItem, component)
	testutil.Exec(t, pool, `INSERT INTO item_unit_conversions(item_code,from_uom,to_uom,factor,created_by) VALUES($1,'UN','KG',2,$2)`, scrapItem, uid)
	if err := repo.ConfigureWarehouseAddress(ctx, 8, "A-01", true); err != nil {
		t.Fatal(err)
	}
	lotCode, address := "SCRAP-1", "A-01"
	destination, err := repo.AddScrapDestination(ctx, &entity.ScrapDestination{ProductionOrderID: orderID, DestinationKind: "ORDER_ITEM", ScrapItemCode: scrapItem, WarehouseID: 8, Lot: &lotCode, Address: &address, ScrapQuantity: decimal.NewFromInt(1), Quantity: decimal.NewFromInt(1), SourceUOM: "UN", ScrapUOM: "KG", DestinationDate: time.Now(), CreatedBy: uid})
	if err != nil {
		t.Fatal(err)
	}
	otherTenantCtx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID + 999999})
	if _, err := repo.UpdateScrapDestination(otherTenantCtx, destination.ID, &entity.ScrapDestination{ProductionOrderID: orderID, ScrapItemCode: scrapItem, WarehouseID: 8, Quantity: decimal.NewFromInt(1), DestinationDate: time.Now(), CreatedBy: uid}); err == nil {
		t.Fatal("cross-tenant scrap update must be rejected")
	}
	updated, err := repo.UpdateScrapDestination(ctx, destination.ID, &entity.ScrapDestination{ProductionOrderID: orderID, DestinationKind: "ORDER_ITEM", ScrapItemCode: scrapItem, WarehouseID: 8, Lot: &lotCode, Address: &address, ScrapQuantity: decimal.RequireFromString("0.5"), Quantity: decimal.RequireFromString("0.5"), SourceUOM: "UN", ScrapUOM: "KG", DestinationDate: time.Now(), CreatedBy: uid})
	if err != nil || updated.ID != destination.ID {
		t.Fatalf("updated destination=%+v err=%v", updated, err)
	}
	var balanceAfterUpdate decimal.Decimal
	if err := pool.QueryRow(ctx, `SELECT quantity FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND mask='' AND warehouse_id=8`, enterpriseID, scrapItem).Scan(&balanceAfterUpdate); err != nil || !balanceAfterUpdate.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("balance after update=%s err=%v", balanceAfterUpdate, err)
	}
	if err := repo.DeleteScrapDestination(ctx, destination.ID, uid); err != nil {
		t.Fatal(err)
	}
	var converted decimal.Decimal
	if err := pool.QueryRow(ctx, `SELECT MAX(quantity) FROM stock_movements WHERE enterprise_id=$1 AND reference_type='PRODUCTION_SCRAP_REVERSED' AND reference_code=$2`, enterpriseID, destination.ID).Scan(&converted); err != nil || !converted.Equal(decimal.NewFromInt(2)) {
		t.Fatalf("converted scrap=%s err=%v", converted, err)
	}
	testutil.Exec(t, pool, `INSERT INTO manufacturing_stock_closed_periods(enterprise_id,period_from,period_to) VALUES($1,CURRENT_DATE,CURRENT_DATE)`, enterpriseID)
	if _, err := repo.AddScrapDestination(ctx, &entity.ScrapDestination{ProductionOrderID: orderID, DestinationKind: "ORDER_ITEM", ScrapItemCode: scrapItem, WarehouseID: 8, Lot: &lotCode, Address: &address, ScrapQuantity: decimal.NewFromInt(1), DestinationDate: time.Now(), CreatedBy: uid}); err == nil {
		t.Fatal("closed stock period must reject scrap destination")
	}
}

func TestPhase4ManualOrderTransfersAutomaticComponentsToLineWarehouse(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(base, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	repo := production_order.NewProductionOrderRepositoryPGX(pool)
	uid := uuid.New()
	parent, child := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO manufacturing_stock_item_controls(enterprise_id,item_code,automatic_issue_type,inventory_group_type,line_warehouse_id) VALUES($1,$2,'TRANSFER','STANDARD',1),($1,$3,'ISSUE','STANDARD',2)`, enterpriseID, parent, child)
	testutil.Exec(t, pool, `INSERT INTO stock_balances(item_code,mask,warehouse_id,quantity,enterprise_id) VALUES($1,'',1,10,$2)`, child, enterpriseID)
	var orderID int64
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM stock_movements WHERE enterprise_id=$1 AND reference_type='PRODUCTION_ORDER_TRANSFER'", enterpriseID)
		pool.Exec(ctx, "DELETE FROM production_orders WHERE id=$1", orderID)
		pool.Exec(ctx, "DELETE FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2", enterpriseID, child)
		pool.Exec(ctx, "DELETE FROM manufacturing_stock_item_controls WHERE enterprise_id=$1 AND item_code=ANY($2)", enterpriseID, []int64{parent, child})
	})
	created, err := repo.CreateWithMaterials(ctx, &entity.ProductionOrder{OrderNumber: testutil.UniqueCode(), ItemCode: parent, PlannedQty: 2, Status: entity.StatusOpen, IsActive: true, CreatedBy: uid}, []*entity.ProductionOrderMaterial{{Kind: entity.MaterialDemand, ItemCode: child, Quantity: decimal.NewFromInt(3), WarehouseID: 1, AutomaticIssue: true, CreatedBy: uid}})
	if err != nil {
		t.Fatal(err)
	}
	orderID = created.ID
	var movements int
	var lineQty decimal.Decimal
	if err := pool.QueryRow(ctx, `SELECT COUNT(*),COALESCE(SUM(CASE WHEN movement_type='TRANSFER_IN' AND warehouse_id=2 THEN quantity ELSE 0 END),0) FROM stock_movements WHERE enterprise_id=$1 AND reference_type='PRODUCTION_ORDER_TRANSFER' AND reference_code=$2`, enterpriseID, orderID).Scan(&movements, &lineQty); err != nil || movements != 2 || !lineQty.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("transfer movements=%d line=%s err=%v", movements, lineQty, err)
	}
	var warehouse int64
	if err := pool.QueryRow(ctx, `SELECT warehouse_id FROM production_order_materials WHERE production_order_id=$1`, orderID).Scan(&warehouse); err != nil || warehouse != 2 {
		t.Fatalf("material warehouse=%d err=%v", warehouse, err)
	}
}
