//go:build integration

package purchase_price_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity"
	priceRepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_price/repository"
	pprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_price"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestIntegrationPurchasePriceTenantResolutionAndAdjustments(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := pprepo.New(q, pool)
	ctx := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(ctx, "SELECT id FROM enterprise ORDER BY id LIMIT 1").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	code, err := repo.NextTableCode(ctx, enterpriseID)
	if err != nil {
		t.Fatal(err)
	}
	var supplier int64
	if err := pool.QueryRow(ctx, "SELECT code FROM suppliers WHERE is_active ORDER BY code LIMIT 1").Scan(&supplier); err != nil {
		t.Skip("integration database has no supplier")
	}
	tbl, err := entity.NewPurchasePriceTable(enterpriseID, code, supplier, "Tabela integração", "BRL", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	created, err := repo.CreateTable(ctx, tbl)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM purchase_price_tables WHERE id=$1", created.ID) })

	item, err := entity.NewPurchasePriceTableItem(created.ID, testutil.UniqueCode(), decimal.RequireFromString("8.500001"))
	if err != nil {
		t.Fatal(err)
	}
	item.SupplierCode = &supplier
	item.Adjustments = []*entity.PriceAdjustment{{Sequence: 1, Kind: "DISCOUNT", CalculationType: "PERCENT", Value: decimal.RequireFromString("2.5")}}
	saved, err := repo.AddItem(ctx, enterpriseID, item)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetItemPrice(ctx, enterpriseID, code, item.ItemCode, &supplier)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Price.Equal(item.Price) {
		t.Fatalf("price=%s want=%s", got.Price, item.Price)
	}
	items, err := repo.ListItems(ctx, enterpriseID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || len(items[0].Adjustments) != 1 || items[0].ID != saved.ID {
		t.Fatalf("unexpected items: %+v", items)
	}
	if _, err = repo.GetTableByCode(ctx, enterpriseID+987654, code); err == nil {
		t.Fatal("cross-tenant lookup unexpectedly succeeded")
	}

	var enterpriseCode int64
	if err = pool.QueryRow(ctx, "SELECT code FROM enterprise WHERE id=$1", enterpriseID).Scan(&enterpriseCode); err != nil {
		t.Fatal(err)
	}
	var orderID, lineID int64
	if err = pool.QueryRow(ctx, `INSERT INTO purchase_orders(order_number,enterprise_code,supplier_code,created_by) VALUES($1,$2,$3,$4) RETURNING code`, testutil.UniqueCode(), enterpriseCode, supplier, uuid.New()).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM purchase_order_items WHERE purchase_order_code=$1", orderID)
		testutil.Exec(t, pool, "DELETE FROM purchase_orders WHERE code=$1", orderID)
	})
	sourceItem := testutil.UniqueCode()
	if err = pool.QueryRow(ctx, `INSERT INTO purchase_order_items(purchase_order_code,item_code,purchase_uom,unit_price) VALUES($1,$2,'CX',12.345678) RETURNING code`, orderID, sourceItem).Scan(&lineID); err != nil {
		t.Fatal(err)
	}
	today := time.Now().Truncate(24 * time.Hour)
	sources, err := repo.ListSourcePrices(ctx, priceRepo.SourceFilter{EnterpriseID: enterpriseID, SupplierCode: &supplier, Start: today.AddDate(0, 0, -1), End: today.AddDate(0, 0, 1), Source: "PURCHASE_ORDER"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, source := range sources {
		if source.SourceID == lineID {
			found = true
		}
	}
	if !found {
		t.Fatalf("purchase order source %d not listed", lineID)
	}
	applied, err := repo.ApplySourcePrices(ctx, enterpriseID, code, false, []priceRepo.ApplySourceSelection{{SourceType: "PURCHASE_ORDER", SourceID: lineID}})
	if err != nil {
		t.Fatal(err)
	}
	if applied != 1 {
		t.Fatalf("applied=%d want=1", applied)
	}
	imported, err := repo.GetItemPrice(ctx, enterpriseID, code, sourceItem, &supplier)
	if err != nil {
		t.Fatal(err)
	}
	if imported.UOM == nil || *imported.UOM != "CX" || !imported.Price.Equal(decimal.RequireFromString("12.345678")) {
		t.Fatalf("unexpected imported price: %+v", imported)
	}
	var entryID, entryItemID int64
	if err = pool.QueryRow(ctx, `INSERT INTO fiscal_entries(numero_nf,serie,modelo,data_emissao,data_entrada,cnpj_emitente,razao_social_emitente,tipo_documento,created_by,supplier_code,enterprise_id) VALUES($1,'1','55',CURRENT_DATE,CURRENT_DATE,'00000000000000','Fornecedor','NFE',$2,$3,$4) RETURNING id`, testutil.UniqueCode(), uuid.New(), supplier, enterpriseID).Scan(&entryID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM fiscal_entries WHERE id=$1", entryID) })
	if err = pool.QueryRow(ctx, `INSERT INTO fiscal_entry_items(fiscal_entry_id,item_code,cfop,quantity,unit_price,total_price,uom) VALUES($1,$2,'1102',1,13.500001,13.500001,'UN') RETURNING id`, entryID, sourceItem).Scan(&entryItemID); err != nil {
		t.Fatal(err)
	}
	both, err := repo.ListSourcePrices(ctx, priceRepo.SourceFilter{EnterpriseID: enterpriseID, TableCode: &code, Start: today.AddDate(0, 0, -1), End: today.AddDate(0, 0, 1), Source: "BOTH"})
	if err != nil {
		t.Fatal(err)
	}
	duplicates := 0
	for _, source := range both {
		if source.ItemCode == sourceItem {
			duplicates++
		}
	}
	if duplicates < 2 {
		t.Fatalf("BOTH should list order and invoice separately, got %d", duplicates)
	}
	applied, err = repo.ApplySourcePrices(ctx, enterpriseID, code, true, []priceRepo.ApplySourceSelection{{SourceType: "ENTRY_INVOICE", SourceID: entryItemID}})
	if err != nil || applied != 1 {
		t.Fatalf("invoice apply=%d err=%v", applied, err)
	}
	imported, err = repo.GetItemPrice(ctx, enterpriseID, code, sourceItem, &supplier)
	if err != nil {
		t.Fatal(err)
	}
	if imported.UOM == nil || *imported.UOM != "UN" || !imported.Price.Equal(decimal.RequireFromString("13.500001")) {
		t.Fatalf("unexpected invoice overwrite: %+v", imported)
	}
}
