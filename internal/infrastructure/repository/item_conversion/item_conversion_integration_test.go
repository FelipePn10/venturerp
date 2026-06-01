//go:build integration

package item_conversion_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
	icrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_conversion"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func TestIntegration_ItemConversion_CRUDAndUpsert(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := icrepo.New(q, pool)
	ctx := context.Background()

	itemCode := testutil.UniqueCode()
	defer testutil.Exec(t, pool, "DELETE FROM item_unit_conversions WHERE item_code = $1", itemCode)

	c, err := entity.NewItemUnitConversion(itemCode, "CX", "UN", 12, uuid.New())
	if err != nil {
		t.Fatalf("NewItemUnitConversion: %v", err)
	}
	if _, err := repo.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, itemCode, "CX", "UN")
	if err != nil || got.Factor != 12 {
		t.Fatalf("Get = %+v err=%v, want factor 12", got, err)
	}
	if list, err := repo.ListByItem(ctx, itemCode); err != nil || len(list) != 1 {
		t.Fatalf("ListByItem = %+v err=%v, want 1", list, err)
	}

	// Upsert: same (item, from, to) updates the factor in place.
	c2, _ := entity.NewItemUnitConversion(itemCode, "CX", "UN", 6, uuid.New())
	if _, err := repo.Create(ctx, c2); err != nil {
		t.Fatalf("Create upsert: %v", err)
	}
	got, _ = repo.Get(ctx, itemCode, "CX", "UN")
	if got.Factor != 6 {
		t.Errorf("upserted factor = %v, want 6", got.Factor)
	}

	// Soft delete → no longer listed.
	if err := repo.Delete(ctx, got.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if list, _ := repo.ListByItem(ctx, itemCode); len(list) != 0 {
		t.Errorf("after delete ListByItem = %v, want empty", list)
	}
}
