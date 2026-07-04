//go:build integration

package cost_uc_test

import (
	"context"
	"math"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cost_uc"
	scentity "github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity"
	standardCostRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/standard_cost"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Rolls up a parent whose BOM has a normal input, a by-product (credit) and a
// fixed-quantity component (amortized over the reference lot):
//
//	material = A(10)×2  +  C(5)×(10÷lot 10)  −  B(8)×1  = 20 + 5 − 8 = 17
func TestIntegration_CostRollup_CoproductAndFixedQty(t *testing.T) {
	q, pool := testutil.Queries(t)
	scRepo := standardCostRepo.New(q)
	uc := cost_uc.New(scRepo)
	ctx := context.Background()
	uid := uuid.New()

	p := testutil.UniqueCode() // parent (manufactured)
	a := testutil.UniqueCode() // normal input
	b := testutil.UniqueCode() // by-product (output)
	c := testutil.UniqueCode() // fixed-qty component
	for _, code := range []int64{p, a, b, c} {
		testutil.Exec(t, pool, "INSERT INTO items (code, warehouse_code, created_by) VALUES ($1,$2,$3)", code, code, uid)
	}
	defer testutil.Exec(t, pool, "DELETE FROM items WHERE code IN ($1,$2,$3,$4)", p, a, b, c)

	// BOM: P → A (2, normal), B (1, co-product), C (10, fixed).
	testutil.Exec(t, pool, "INSERT INTO item_structures (parent_code, child_code, quantity, sequence, created_by) VALUES ($1,$2,2,1,$3)", p, a, uid)
	testutil.Exec(t, pool, "INSERT INTO item_structures (parent_code, child_code, quantity, sequence, is_coproduct, created_by) VALUES ($1,$2,1,2,TRUE,$3)", p, b, uid)
	testutil.Exec(t, pool, "INSERT INTO item_structures (parent_code, child_code, quantity, sequence, is_fixed_qty, created_by) VALUES ($1,$2,10,3,TRUE,$3)", p, c, uid)
	defer testutil.Exec(t, pool, "DELETE FROM item_structures WHERE parent_code = $1", p)

	// Purchase costs for the leaves.
	for code, cost := range map[int64]float64{a: 10, b: 8, c: 5} {
		if _, err := scRepo.UpsertItemPurchaseCost(ctx, &scentity.ItemPurchaseCost{ItemCode: code, UnitCost: cost, Currency: "BRL", UpdatedBy: uid}); err != nil {
			t.Fatalf("UpsertItemPurchaseCost(%d): %v", code, err)
		}
	}
	defer testutil.Exec(t, pool, "DELETE FROM item_purchase_costs WHERE item_code IN ($1,$2,$3)", a, b, c)
	defer testutil.Exec(t, pool, "DELETE FROM item_standard_costs WHERE item_code = $1", p)
	defer testutil.Exec(t, pool, "DELETE FROM cost_rollup_log WHERE item_code = $1", p)

	res, err := uc.RollUp(ctx, request.CostRollupDTO{ItemCode: p, LotSize: 10, CalculatedBy: uid.String()})
	if err != nil {
		t.Fatalf("RollUp: %v", err)
	}
	if math.Abs(res.MaterialCost-17) > 0.01 {
		t.Fatalf("material_cost = %.4f, want 17 (20 + 5 − 8 by-product credit)", res.MaterialCost)
	}
	t.Logf("material_cost=%.2f — by-product credit + fixed-qty amortization OK", res.MaterialCost)
}
