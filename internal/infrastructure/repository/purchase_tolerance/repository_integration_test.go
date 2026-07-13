//go:build integration

package purchase_tolerance_test

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_tolerance/entity"
	repoimpl "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_tolerance"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"testing"
)

func TestIntegrationSupplierToleranceOverridesGeneric(t *testing.T) {
	_, pool := testutil.Queries(t)
	repo := repoimpl.New(pool)
	ctx := context.Background()
	var e int64
	if err := pool.QueryRow(ctx, "SELECT id FROM enterprise ORDER BY id LIMIT 1").Scan(&e); err != nil {
		t.Fatal(err)
	}
	supplier := testutil.UniqueCode()
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM purchase_order_tolerances WHERE enterprise_id=$1 AND (supplier_code=$2 OR supplier_code IS NULL) AND created_by=$3", e, supplier, testUser)
	})
	g, _ := entity.New(e, entity.ToleranceQuantity, entity.AppliesAll, decimal.Zero, nil, decimal.NewFromInt(10), entity.ValuePercent, nil, entity.ActionWarn, testUser)
	if _, err := repo.Save(ctx, g); err != nil {
		t.Fatal(err)
	}
	s, _ := entity.New(e, entity.ToleranceQuantity, entity.AppliesReceivingNotice, decimal.Zero, nil, decimal.NewFromInt(2), entity.ValuePercent, &supplier, entity.ActionBlock, testUser)
	if _, err := repo.Save(ctx, s); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Resolve(ctx, e, &supplier, entity.ToleranceQuantity, entity.AppliesReceivingNotice, decimal.NewFromInt(100))
	if err != nil {
		t.Fatal(err)
	}
	if got.SupplierCode == nil || got.Action != entity.ActionBlock {
		t.Fatalf("specific rule not selected: %+v", got)
	}
}

var testUser = uuid.MustParse("00000000-0000-0000-0000-000000000237")
