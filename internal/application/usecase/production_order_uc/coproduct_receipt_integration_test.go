//go:build integration

package production_order_uc_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	prodrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	structrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

type fakeCompleteAuth struct{ ports.AuthService }

func (fakeCompleteAuth) CanUpdateSalesOrder(context.Context) bool { return true }

// Completing an OF receives the item's BOM co-products / returnable scrap into
// stock: an IN movement of coproduct_qty × produced.
func TestIntegration_CompleteReceivesCoproductScrap(t *testing.T) {
	q, pool := testutil.Queries(t)
	ctx := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(ctx, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	uid := uuid.New()

	finished := testutil.UniqueCode()
	scrap := testutil.UniqueCode()
	for _, code := range []int64{finished, scrap} {
		testutil.Exec(t, pool, "INSERT INTO items (code, warehouse_code, created_by) VALUES ($1,$2,$3)", code, code, uid)
	}
	defer testutil.Exec(t, pool, "DELETE FROM items WHERE code IN ($1,$2)", finished, scrap)

	// BOM: finished → scrap (co-product, 0.5 per unit).
	testutil.Exec(t, pool, "INSERT INTO item_structures (parent_code, child_code, quantity, sequence, is_coproduct, created_by) VALUES ($1,$2,0.5,1,TRUE,$3)", finished, scrap, uid)
	defer testutil.Exec(t, pool, "DELETE FROM item_structures WHERE parent_code = $1", finished)

	// Production order that produced 100 units.
	var poID int64
	if err := pool.QueryRow(ctx,
		"INSERT INTO production_orders (order_number,item_code,planned_qty,produced_qty,status,created_by,enterprise_id) VALUES ($1,$2,$3,$4,'IN_PROGRESS',$5,$6) RETURNING id",
		testutil.UniqueCode(), finished, 100, 100, uid, enterpriseID).Scan(&poID); err != nil {
		t.Fatalf("seed production_order: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM production_orders WHERE id = $1", poID)
	defer testutil.Exec(t, pool, "DELETE FROM stock_movements WHERE reference_code = $1 AND reference_type = 'PRODUCTION_ORDER'", poID)

	wh := int64(1)
	uc := &production_order_uc.CompleteProductionOrderUseCase{
		Repo:      prodrepo.NewProductionOrderRepositoryPGX(pool),
		Auth:      fakeCompleteAuth{},
		StockRepo: stockrepo.NewStockRepositorySQLC(pool),
		Structure: structrepo.NewItemStructureRepository(q),
	}
	if _, err := uc.Execute(ctx, request.CompleteProductionOrderDTO{ID: poID, WarehouseID: &wh}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	movs, err := uc.StockRepo.ListMovementsByItem(ctx, scrap)
	if err != nil {
		t.Fatalf("ListMovementsByItem: %v", err)
	}
	var received float64
	for _, m := range movs {
		if m.MovementType == "IN" {
			received += m.Quantity
		}
	}
	if received != 50 {
		t.Fatalf("scrap received = %v, want 50 (0.5 × 100)", received)
	}
}
