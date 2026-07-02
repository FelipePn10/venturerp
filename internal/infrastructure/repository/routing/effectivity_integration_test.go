//go:build integration

package routing_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routingrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// GetRouteForItem must return the route effective TODAY, ignoring expired and
// not-yet-effective (future) revisions.
func TestIntegration_Routing_EffectivitySelection(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := routingrepo.New(q)
	ctx := context.Background()
	uid := uuid.New()

	itemCode := testutil.UniqueCode()
	testutil.Exec(t, pool, "INSERT INTO items (code, warehouse_code, created_by) VALUES ($1,$2,$3)", itemCode, itemCode, uid)
	defer testutil.Exec(t, pool, "DELETE FROM items WHERE code = $1", itemCode)
	defer testutil.Exec(t, pool, "DELETE FROM manufacturing_routes WHERE item_code = $1", itemCode)

	now := time.Now()
	expiredTo := now.AddDate(0, 0, -1)   // yesterday
	pastFrom := now.AddDate(0, 0, -30)   // a month ago
	futureFrom := now.AddDate(0, 0, 10)  // next week+

	// Expired revision (alt 1).
	rExpired, _ := entity.NewManufacturingRoute(testutil.UniqueCode(), itemCode, nil, 1, ptrStr("REV A (expirada)"), true, &pastFrom, &expiredTo, uid)
	if _, err := repo.CreateRoute(ctx, rExpired); err != nil {
		t.Fatalf("create expired: %v", err)
	}
	// Currently-effective revision (alt 2), open-ended.
	rCurrent, _ := entity.NewManufacturingRoute(testutil.UniqueCode(), itemCode, nil, 2, ptrStr("REV B (vigente)"), true, &pastFrom, nil, uid)
	current, err := repo.CreateRoute(ctx, rCurrent)
	if err != nil {
		t.Fatalf("create current: %v", err)
	}
	// Future revision (alt 3).
	rFuture, _ := entity.NewManufacturingRoute(testutil.UniqueCode(), itemCode, nil, 3, ptrStr("REV C (futura)"), true, &futureFrom, nil, uid)
	if _, err := repo.CreateRoute(ctx, rFuture); err != nil {
		t.Fatalf("create future: %v", err)
	}

	got, err := repo.GetRouteForItem(ctx, itemCode, "")
	if err != nil {
		t.Fatalf("GetRouteForItem: %v", err)
	}
	if got.ID != current.ID {
		t.Fatalf("selected route id=%d (alt %d), want the current one id=%d (alt 2)", got.ID, got.Alternative, current.ID)
	}
	if got.ValidTo != nil {
		t.Errorf("current route should be open-ended, got valid_to=%v", got.ValidTo)
	}
}

func ptrStr(s string) *string { return &s }
