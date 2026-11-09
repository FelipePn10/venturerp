//go:build integration

package item_conversion_test

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
	icrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_conversion"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func TestIntegration_ItemConversion_CRUDAndUpdate(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := icrepo.New(q, pool)
	// A conversão é da empresa do item: o teste roda numa empresa e cria o item nela.
	ctx := testutil.TenantContext(t, pool)

	itemCode := testutil.UniqueCode()
	testutil.SeedItem(t, pool, ctx, itemCode, testutil.Actor(t, pool))
	defer testutil.Exec(t, pool, "DELETE FROM items WHERE code = $1", itemCode)
	defer testutil.Exec(t, pool, "DELETE FROM item_unit_conversions WHERE item_code = $1", itemCode)

	c, err := entity.NewItemUnitConversion(itemCode, "", "CX", "UN", decimal.NewFromFloat(12), decimal.NewFromFloat(0), decimal.NewFromFloat(0), "VALUE", uuid.New())
	if err != nil {
		t.Fatalf("NewItemUnitConversion: %v", err)
	}
	if _, err := repo.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, itemCode, "CX", "UN")
	if err != nil || !got.Factor.Equal(decimal.NewFromFloat(12)) {
		t.Fatalf("Get = %+v err=%v, want factor 12", got, err)
	}
	if list, err := repo.ListByItem(ctx, itemCode); err != nil || len(list) != 1 {
		t.Fatalf("ListByItem = %+v err=%v, want 1", list, err)
	}

	// O mesmo fato vigente não é gravado duas vezes (antes o Create sobrescrevia
	// o fator em silêncio): quem muda o fator usa Update, que grava o histórico.
	c2, _ := entity.NewItemUnitConversion(itemCode, "", "CX", "UN", decimal.NewFromFloat(6), decimal.NewFromFloat(0), decimal.NewFromFloat(0), "VALUE", uuid.New())
	if _, err := repo.Create(ctx, c2); err == nil {
		t.Fatal("Create do mesmo par vigente deveria ser recusado")
	}
	if err := got.SetFraction(decimal.NewFromInt(1), decimal.NewFromInt(6)); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Update(ctx, got, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = repo.Get(ctx, itemCode, "CX", "UN")
	if !got.Factor.Equal(decimal.NewFromFloat(6)) {
		t.Errorf("fator alterado = %v, want 6", got.Factor)
	}

	// Soft delete → no longer listed.
	if err := repo.Delete(ctx, got.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if list, _ := repo.ListByItem(ctx, itemCode); len(list) != 0 {
		t.Errorf("after delete ListByItem = %v, want empty", list)
	}
}
