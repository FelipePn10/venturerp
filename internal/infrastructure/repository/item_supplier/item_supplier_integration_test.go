//go:build integration

package item_supplier_test

import (
	"context"
	"fmt"
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
	actor := testutil.Actor(t, pool)
	// O vínculo só aceita item e fornecedor da própria empresa.
	testutil.Exec(t, pool, "INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,$1,$2,$3)", item, actor, e)
	testutil.Exec(t, pool, "INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'FORNECEDOR',$2,$3,$4)", supplier, fmt.Sprintf("%014d", supplier%100_000_000_000_000), actor, e)
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM item_preferred_suppliers WHERE enterprise_id=$1 AND item_code=$2", e, item)
		testutil.Exec(t, pool, "DELETE FROM suppliers WHERE code=$1", supplier)
		testutil.Exec(t, pool, "DELETE FROM items WHERE code=$1", item)
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

// TestItemSupplierRefusesOtherTenantRecords: código de item e de fornecedor são
// únicos no banco inteiro, e o vínculo item×fornecedor aceitava o item ou o
// fornecedor de outra empresa.
func TestItemSupplierRefusesOtherTenantRecords(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := repoimpl.New(q, pool)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		_, _ = pool.Exec(base, "DELETE FROM item_preferred_suppliers WHERE enterprise_id = ANY($1)", ids)
		_, _ = pool.Exec(base, "DELETE FROM suppliers WHERE enterprise_id = ANY($1)", ids)
		_, _ = pool.Exec(base, "DELETE FROM items WHERE enterprise_id = ANY($1)", ids)
	})
	semeia := func(e testutil.Empresa) (int64, int64) {
		item, supplier := testutil.UniqueCode(), testutil.UniqueCode()
		testutil.Exec(t, pool, "INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,$1,$2,$3)", item, actor, e.ID)
		testutil.Exec(t, pool, "INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'FORNECEDOR',$2,$3,$4)", supplier, fmt.Sprintf("%014d", supplier%100_000_000_000_000), actor, e.ID)
		return item, supplier
	}
	itemA, fornA := semeia(a)
	itemB, fornB := semeia(b)

	casos := []struct {
		nome           string
		item, supplier int64
	}{
		{"item da B", itemB, fornA},
		{"fornecedor da B", itemA, fornB},
	}
	for _, c := range casos {
		v, _ := entity.NewItemPreferredSupplier(a.ID, c.item, c.supplier, "", 1, actor)
		if _, err := repo.Upsert(a.Ctx, v); err == nil {
			t.Fatalf("a empresa A gravou vínculo com o %s", c.nome)
		}
	}
	v, _ := entity.NewItemPreferredSupplier(a.ID, itemA, fornA, "", 1, actor)
	if _, err := repo.Upsert(a.Ctx, v); err != nil {
		t.Fatalf("a A não gravou o vínculo com os próprios cadastros: %v", err)
	}
}
