//go:build integration

package stock_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func TestStockMovementAndBalanceAreTenantIsolated(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	var first int64
	if err := pool.QueryRow(base, "SELECT MIN(id) FROM enterprise").Scan(&first); err != nil {
		t.Fatal(err)
	}
	var enterpriseCode int32
	if err := pool.QueryRow(base, "SELECT COALESCE(MAX(code),0)+1 FROM enterprise").Scan(&enterpriseCode); err != nil {
		t.Fatal(err)
	}
	var second int64
	if err := pool.QueryRow(base, "INSERT INTO enterprise(code,name) VALUES($1,$2) RETURNING id", enterpriseCode, "Tenant stock test").Scan(&second); err != nil {
		t.Fatal(err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM enterprise WHERE id=$1", second)
	ctx1 := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: first})
	ctx2 := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: second})
	repo := stockrepo.NewStockRepositorySQLC(pool)
	item := testutil.UniqueCode()
	uid := uuid.New()
	for _, entry := range []struct {
		ctx context.Context
		qty string
	}{{ctx1, "1.123456"}, {ctx2, "9.654321"}} {
		q := decimal.RequireFromString(entry.qty)
		f, _ := q.Float64()
		_, err := repo.CreateMovement(entry.ctx, &stockentity.StockMovement{ItemCode: item, WarehouseID: 1, MovementType: "IN", Quantity: f, ExactQuantity: q, CreatedBy: uid})
		if err != nil {
			t.Fatal(err)
		}
	}
	defer testutil.Exec(t, pool, "DELETE FROM stock_movements WHERE item_code=$1", item)
	defer testutil.Exec(t, pool, "DELETE FROM stock_balances WHERE item_code=$1", item)
	b1, err := repo.GetBalance(ctx1, item, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := repo.GetBalance(ctx2, item, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if decimal.NewFromFloat(b1.Quantity).StringFixed(6) != "1.123456" || decimal.NewFromFloat(b2.Quantity).StringFixed(6) != "9.654321" {
		t.Fatalf("tenant balances leaked or lost precision: %v / %v", b1.Quantity, b2.Quantity)
	}
}
