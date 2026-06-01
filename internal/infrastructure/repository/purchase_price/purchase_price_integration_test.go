//go:build integration

package purchase_price_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity"
	pprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_price"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func TestIntegration_PurchasePrice_Resolution(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := pprepo.New(q, pool)
	ctx := context.Background()

	code, err := repo.NextTableCode(ctx)
	if err != nil {
		t.Fatalf("NextTableCode: %v", err)
	}
	tbl, err := entity.NewPurchasePriceTable(code, "Tabela Integração", "BRL", uuid.New())
	if err != nil {
		t.Fatalf("NewPurchasePriceTable: %v", err)
	}
	created, err := repo.CreateTable(ctx, tbl)
	if err != nil {
		t.Fatalf("CreateTable: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM purchase_price_tables WHERE code = $1", code)

	const itemCode = int64(123456)
	supplierA := int64(7777)

	// Generic price (any supplier) = 10.00
	generic, _ := entity.NewPurchasePriceTableItem(created.ID, itemCode, 10)
	if _, err := repo.AddItem(ctx, generic); err != nil {
		t.Fatalf("AddItem generic: %v", err)
	}
	// Supplier-specific price for supplierA = 8.50
	specific, _ := entity.NewPurchasePriceTableItem(created.ID, itemCode, 8.5)
	specific.SupplierCode = &supplierA
	if _, err := repo.AddItem(ctx, specific); err != nil {
		t.Fatalf("AddItem specific: %v", err)
	}

	// With supplierA → must prefer the supplier-specific price (8.50).
	got, err := repo.GetItemPrice(ctx, code, itemCode, &supplierA)
	if err != nil {
		t.Fatalf("GetItemPrice(supplierA): %v", err)
	}
	if got.Price != 8.5 {
		t.Errorf("supplier-specific price = %v, want 8.50", got.Price)
	}

	// Without supplier → generic (10.00).
	got, err = repo.GetItemPrice(ctx, code, itemCode, nil)
	if err != nil {
		t.Fatalf("GetItemPrice(nil): %v", err)
	}
	if got.Price != 10 {
		t.Errorf("generic price = %v, want 10.00", got.Price)
	}

	// A different supplier (no specific row) → falls back to generic (10.00).
	other := int64(8888)
	got, err = repo.GetItemPrice(ctx, code, itemCode, &other)
	if err != nil {
		t.Fatalf("GetItemPrice(other): %v", err)
	}
	if got.Price != 10 {
		t.Errorf("fallback price = %v, want 10.00", got.Price)
	}

	// Upsert: re-adding the generic row updates the price in place.
	generic2, _ := entity.NewPurchasePriceTableItem(created.ID, itemCode, 11)
	if _, err := repo.AddItem(ctx, generic2); err != nil {
		t.Fatalf("AddItem upsert: %v", err)
	}
	got, _ = repo.GetItemPrice(ctx, code, itemCode, nil)
	if got.Price != 11 {
		t.Errorf("upserted generic price = %v, want 11.00", got.Price)
	}
}
