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
	// Código único: MAX(code)+1 colidia com a empresa criada em paralelo por
	// outro pacote (enterprise_code_key).
	enterpriseCode := int32(1_620_000_000 + testutil.UniqueCode()%10_000_000)
	var second int64
	if err := pool.QueryRow(base, "INSERT INTO enterprise(code,name) VALUES($1,$2) RETURNING id", enterpriseCode, "Tenant stock test").Scan(&second); err != nil {
		t.Fatal(err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM enterprise WHERE id=$1", second)
	ctx1 := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: first})
	ctx2 := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: second})
	repo := stockrepo.NewStockRepositorySQLC(pool)
	ator := testutil.Actor(t, pool)
	type dono struct {
		ctx         context.Context
		empresa     int64
		item, almox int64
		qty         string
	}
	donos := []*dono{{ctx: ctx1, empresa: first, qty: "1.123456"}, {ctx: ctx2, empresa: second, qty: "9.654321"}}
	for _, d := range donos {
		d.item = testutil.UniqueCode()
		if err := pool.QueryRow(base,
			`INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
			 VALUES($1,$1,$2,'INTERNO','NORMAL',true,true,$3) RETURNING id`, "TEN-"+uuid.NewString()[:8], ator, d.empresa).Scan(&d.almox); err != nil {
			t.Fatal(err)
		}
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,$2,$3,$4)`, d.item, d.almox, ator, d.empresa)
		defer testutil.Exec(t, pool, "DELETE FROM warehouse WHERE id=$1", d.almox)
		defer testutil.Exec(t, pool, "DELETE FROM items WHERE code=$1", d.item)
		defer testutil.Exec(t, pool, "DELETE FROM stock_balances WHERE item_code=$1", d.item)
		defer testutil.Exec(t, pool, "DELETE FROM stock_movements WHERE item_code=$1", d.item)
		if _, err := repo.CreateMovement(d.ctx, &stockentity.StockMovement{ItemCode: d.item, WarehouseID: d.almox, MovementType: "IN", Quantity: decimal.RequireFromString(d.qty), CreatedBy: ator}); err != nil {
			t.Fatal(err)
		}
	}
	for i, d := range donos {
		b, err := repo.GetBalance(d.ctx, d.item, "", d.almox)
		if err != nil {
			t.Fatal(err)
		}
		if b.Quantity.StringFixed(6) != d.qty {
			t.Fatalf("saldo perdeu precisão: %v", b.Quantity)
		}
		outra := donos[1-i]
		if _, err := repo.GetBalance(d.ctx, outra.item, "", outra.almox); err == nil {
			t.Fatal("empresa leu o saldo da outra")
		}
		if _, err := repo.CreateMovement(d.ctx, &stockentity.StockMovement{ItemCode: d.item, WarehouseID: outra.almox, MovementType: "IN", Quantity: decimal.NewFromInt(1), CreatedBy: ator}); err == nil {
			t.Fatal("movimento entrou no almoxarifado da outra empresa")
		}
	}
}
