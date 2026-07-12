//go:build integration

package production_order_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func TestMaterialControl_SubstitutionWMSAndLotRules(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(base, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	uid := uuid.New()
	orderItem, component := testutil.UniqueCode(), testutil.UniqueCode()
	var orderID int64
	if err := pool.QueryRow(ctx, `INSERT INTO production_orders
		(order_number,item_code,planned_qty,status,created_by,enterprise_id)
		VALUES ($1,$2,10,'RELEASED',$3,$4) RETURNING id`, testutil.UniqueCode(), orderItem, uid, enterpriseID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM production_orders WHERE id=$1", orderID)
	repo := productionrepo.NewProductionOrderRepositoryPGX(pool)
	candidates, err := repo.ListDeliveryCandidates(ctx, production_order_uc.DeliveryCandidateFilter{ItemFrom: &orderItem, ItemTo: &orderItem})
	if err != nil || len(candidates) != 1 || candidates[0].ID != orderID {
		t.Fatalf("delivery candidate filter failed: %+v err=%v", candidates, err)
	}
	testutil.Exec(t, pool, `INSERT INTO planning_params
		(param_number,param_key,value,description,updated_by,enterprise_id)
		VALUES (66,'VALIDA_QUANTIDADE_RETRABALHO','S','integration test',$2,$1)`, enterpriseID, uid)
	defer testutil.Exec(t, pool, "DELETE FROM planning_params WHERE enterprise_id=$1 AND param_number=66", enterpriseID)
	if _, err := repo.AddMaterial(ctx, &entity.ProductionOrderMaterial{ProductionOrderID: orderID,
		Kind: entity.MaterialDemand, ItemCode: orderItem, Quantity: decimal.NewFromInt(9), WarehouseID: 1, CreatedBy: uid}); err == nil {
		t.Fatal("parameter 66 must reject a rework demand with a quantity different from the order")
	}
	material, err := repo.AddMaterial(ctx, &entity.ProductionOrderMaterial{ProductionOrderID: orderID,
		Kind: entity.MaterialDemand, ItemCode: component, Quantity: decimal.NewFromInt(10), WarehouseID: 1, CreatedBy: uid})
	if err != nil {
		t.Fatal(err)
	}

	replacements, err := repo.ReplaceMaterial(ctx, material.ID, []entity.MaterialSubstitution{
		{ItemCode: component + 1, Quantity: decimal.NewFromInt(4), WarehouseID: 1},
		{ItemCode: component + 2, Quantity: decimal.NewFromInt(3), WarehouseID: 1},
	}, uid)
	if err != nil || len(replacements) != 2 {
		t.Fatalf("replacement failed: count=%d err=%v", len(replacements), err)
	}
	materials, err := repo.ListMaterials(ctx, orderID, entity.MaterialDemand)
	if err != nil {
		t.Fatal(err)
	}
	if len(materials) != 3 || !materials[0].Quantity.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("original must remain with quantity 3: %+v", materials)
	}

	testutil.Exec(t, pool, `INSERT INTO production_order_wms_requests
		(production_order_material_id,enterprise_id,status,external_reference) VALUES ($1,$2,'SENT',$3)`, material.ID, enterpriseID, uuid.NewString())
	if err := repo.DeleteMaterial(ctx, material.ID); err == nil {
		t.Fatal("WMS-linked material deletion must be blocked")
	}

	testutil.Exec(t, pool, `INSERT INTO stock_lot_balances
		(item_code,mask,warehouse_id,lot,quantity,enterprise_id) VALUES ($1,'',1,'LOT-A',5,$2)`, component+1, enterpriseID)
	defer testutil.Exec(t, pool, "DELETE FROM stock_lot_balances WHERE enterprise_id=$1 AND item_code=$2", enterpriseID, component+1)
	tooMuch := []entity.LotAllocation{{WarehouseID: 1, Lot: "LOT-A", Quantity: decimal.NewFromInt(5)}}
	// Replacement quantity is only four, even though the lot has five.
	if _, err := repo.AllocateLots(ctx, replacements[0].ID, "REQUISITION", tooMuch, uid); err == nil {
		t.Fatal("allocation above material need must be blocked")
	}
	allocated, err := repo.AllocateLots(ctx, replacements[0].ID, "REQUISITION",
		[]entity.LotAllocation{{WarehouseID: 1, Lot: "LOT-A", Quantity: decimal.NewFromInt(4)}}, uid)
	if err != nil || len(allocated) != 1 || !allocated[0].Quantity.Equal(decimal.NewFromInt(4)) {
		t.Fatalf("valid lot allocation failed: %+v err=%v", allocated, err)
	}
	var secondOrderID int64
	if err := pool.QueryRow(ctx, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id)
		VALUES($1,$2,10,'RELEASED',$3,$4) RETURNING id`, testutil.UniqueCode(), orderItem, uid, enterpriseID).Scan(&secondOrderID); err != nil {
		t.Fatal(err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM production_orders WHERE id=$1", secondOrderID)
	secondMaterial, err := repo.AddMaterial(ctx, &entity.ProductionOrderMaterial{ProductionOrderID: secondOrderID, Kind: entity.MaterialDemand, ItemCode: component + 1, Quantity: decimal.NewFromInt(3), WarehouseID: 1, CreatedBy: uid})
	if err != nil {
		t.Fatal(err)
	}
	intermediate := int64(1)
	if _, err := repo.UpsertWMSSettings(ctx, entity.WMSWarehouseSettings{WarehouseID: 99, IsWMS: true, IntermediateOutWarehouseID: &intermediate}); err != nil {
		t.Fatal(err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM warehouse_wms_settings WHERE enterprise_id=$1 AND warehouse_id=99", enterpriseID)
	testutil.Exec(t, pool, "UPDATE production_order_materials SET warehouse_id=99 WHERE id=$1", secondMaterial.ID)
	if _, err := repo.AllocateLots(ctx, secondMaterial.ID, "REQUISITION", []entity.LotAllocation{{WarehouseID: 1, Lot: "LOT-A", Quantity: decimal.NewFromInt(1)}}, uid); err != nil {
		t.Fatalf("WMS intermediate allocation failed: %v", err)
	}
	batch, err := repo.AllocateLotsBatch(ctx, []int64{secondMaterial.ID, replacements[0].ID}, "REQUISITION", []entity.LotAllocation{{WarehouseID: 1, Lot: "LOT-A", Quantity: decimal.NewFromInt(5)}}, uid)
	if err != nil || len(batch) != 2 || !batch[0].Quantity.Add(batch[1].Quantity).Equal(decimal.NewFromInt(5)) {
		t.Fatalf("batch allocation failed: %+v err=%v", batch, err)
	}
	testutil.Exec(t, pool, "UPDATE production_orders SET scrapped_qty=2 WHERE id=$1", orderID)
	scrapItem := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO manufacturing_stock_item_controls(enterprise_id,item_code,stock_uom,inventory_group_type) VALUES($1,$2,'UN','SECONDARY_MATERIAL')`, enterpriseID, scrapItem)
	defer testutil.Exec(t, pool, "DELETE FROM manufacturing_stock_item_controls WHERE enterprise_id=$1 AND item_code=$2", enterpriseID, scrapItem)
	destination, err := repo.AddScrapDestination(ctx, &entity.ScrapDestination{ProductionOrderID: orderID, ScrapItemCode: scrapItem, WarehouseID: 8, Quantity: decimal.RequireFromString("1.25"), DestinationDate: time.Now(), CreatedBy: uid})
	if err != nil || destination.ID == 0 {
		t.Fatalf("scrap destination failed: %+v err=%v", destination, err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM stock_movements WHERE enterprise_id=$1 AND item_code=$2", enterpriseID, scrapItem)
	defer testutil.Exec(t, pool, "DELETE FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2", enterpriseID, scrapItem)
	if _, err := repo.AddScrapDestination(ctx, &entity.ScrapDestination{ProductionOrderID: orderID, ScrapItemCode: scrapItem, WarehouseID: 8, Quantity: decimal.NewFromInt(1), DestinationDate: time.Now(), CreatedBy: uid}); err == nil {
		t.Fatal("scrap destination above scrapped quantity must be blocked")
	}
}

func TestParameter45_BlocksOrderReportingWithIssueAtRelease(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(base, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	itemCode := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO items
		(code,warehouse_code,production_reporting_type,material_issue_timing,created_by)
		VALUES ($1,$1,'ORDER','REGISTRATION_RELEASE',$2)`, itemCode, uuid.New())
	defer testutil.Exec(t, pool, "DELETE FROM items WHERE code=$1", itemCode)
	testutil.Exec(t, pool, "UPDATE planning_params SET value='S' WHERE enterprise_id=$1 AND param_number=45", enterpriseID)
	defer testutil.Exec(t, pool, "UPDATE planning_params SET value='N' WHERE enterprise_id=$1 AND param_number=45", enterpriseID)
	if err := productionrepo.NewProductionOrderRepositoryPGX(pool).ValidateProductionRelease(ctx, itemCode); err == nil {
		t.Fatal("parameter 45 must block this production release")
	}
}
