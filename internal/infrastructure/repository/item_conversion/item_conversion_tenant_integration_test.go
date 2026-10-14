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

// TestItemConversionIsIsolatedByTenant: a conversão de unidade é da empresa do
// item; outra empresa não a cria, não a lê nem a apaga.
func TestItemConversionIsIsolatedByTenant(t *testing.T) {
	q, pool := testutil.Queries(t)
	item := int64(1_780_000_000 + testutil.UniqueCode()%10_000_000)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(context.Background(), `DELETE FROM item_unit_conversions WHERE item_code=$1`, item)
		_, _ = pool.Exec(context.Background(), `DELETE FROM items WHERE code=$1`, item)
	})
	testutil.SeedItem(t, pool, a.Ctx, item, testutil.Actor(t, pool))
	repo := icrepo.New(q, pool)
	c, err := entity.NewItemUnitConversion(item, "", "CX", "UN", 12, 0, 0, "VALUE", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(b.Ctx, c); err == nil {
		t.Fatal("empresa B criou conversão para item da A")
	}
	criada, err := repo.Create(a.Ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(b.Ctx, item, "CX", "UN"); err == nil {
		t.Fatal("empresa B leu a conversão do item da A")
	}
	if l, err := repo.ListByItem(b.Ctx, item); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d conversão(ões) da A (%v)", len(l), err)
	}
	if err := repo.Delete(b.Ctx, criada.ID); err == nil {
		t.Fatal("empresa B apagou a conversão da A")
	}
	if _, err := repo.Get(a.Ctx, item, "CX", "UN"); err != nil {
		t.Fatalf("a conversão da A sumiu: %v", err)
	}
}
