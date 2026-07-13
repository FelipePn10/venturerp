//go:build integration

package item_supplier_test

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity"
	repoimpl "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_supplier"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/google/uuid"
	"testing"
)

func TestIntegrationItemSupplierOccurrencesAndPreference(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := repoimpl.New(q, pool)
	ctx := context.Background()
	var e int64
	if err := pool.QueryRow(ctx, "SELECT id FROM enterprise ORDER BY id LIMIT 1").Scan(&e); err != nil {
		t.Fatal(err)
	}
	item, supplier := testutil.UniqueCode(), testutil.UniqueCode()
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM item_preferred_suppliers WHERE enterprise_id=$1 AND item_code=$2", e, item)
	})
	a, _ := entity.NewItemPreferredSupplier(e, item, supplier, "A", 1, uuid.New())
	a.IsPreferred = true
	if _, err := repo.Upsert(ctx, a); err != nil {
		t.Fatal(err)
	}
	b, _ := entity.NewItemPreferredSupplier(e, item, supplier, "B", 2, uuid.New())
	if _, err := repo.Upsert(ctx, b); err != nil {
		t.Fatal(err)
	}
	list, err := repo.ListByItem(ctx, e, item)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("occurrences=%d want=2", len(list))
	}
	preferred, err := repo.GetPreferred(ctx, e, item)
	if err != nil || preferred.Mask != "A" {
		t.Fatalf("preferred=%+v err=%v", preferred, err)
	}
	foreign, err := repo.ListByItem(ctx, e+987654, item)
	if err != nil || len(foreign) != 0 {
		t.Fatalf("tenant leak: %+v %v", foreign, err)
	}
}
