//go:build integration

package stock

import (
	"context"
	"fmt"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func TestReservationAndLotCreationAreTenantAware(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := uuid.New()
	code := int64(1_000_000_000 + testutil.UniqueCode()%500_000_000)
	var enterpriseID, warehouseID int64
	testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password) VALUES($1,'Operacional',$2,'x')`, actor, actor.String()+"@test.local")
	if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Operacional',$2) RETURNING id`, code, actor).Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed) VALUES($1,'Operacional',$2,'NORMAL','INTERNO',TRUE,TRUE) RETURNING id`, fmt.Sprintf("W-%d", code), actor).Scan(&warehouseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	repo := NewStockRepositorySQLC(pool)
	itemCode := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO stock_balances(enterprise_id,item_code,mask,warehouse_id,quantity) VALUES($1,$2,'',$3,10)`, enterpriseID, itemCode, warehouseID)

	t.Cleanup(func() {
		_, _ = pool.Exec(base, `DELETE FROM stock_reservations WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(base, `DELETE FROM stock_lots WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(base, `DELETE FROM stock_balances WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(base, `DELETE FROM warehouse WHERE id=$1`, warehouseID)
		_, _ = pool.Exec(base, `DELETE FROM enterprise WHERE id=$1`, enterpriseID)
		_, _ = pool.Exec(base, `DELETE FROM users WHERE id=$1`, actor)
	})

	reservation, err := repo.CreateReservation(ctx, &stockentity.StockReservation{ItemCode: itemCode, WarehouseID: warehouseID, Quantity: 2, ReferenceType: "TEST", ReferenceCode: code, Status: "ACTIVE", CreatedBy: actor})
	if err != nil || reservation.ID == 0 {
		t.Fatalf("CreateReservation() = %+v, %v", reservation, err)
	}
	lot, err := repo.UpsertLot(ctx, &stockentity.StockLot{ItemCode: itemCode, Mask: "A", Lot: "LOTE-1", CreatedBy: actor})
	if err != nil || lot.ID == 0 {
		t.Fatalf("UpsertLot() = %+v, %v", lot, err)
	}
	if _, err = repo.UpsertLot(ctx, &stockentity.StockLot{ItemCode: itemCode, Mask: "B", Lot: "LOTE-1", CreatedBy: actor}); err != nil {
		t.Fatalf("mesmo lote em máscara diferente: %v", err)
	}
	var count int
	if err = pool.QueryRow(base, `SELECT COUNT(*) FROM stock_lots WHERE enterprise_id=$1 AND item_code=$2 AND lot='LOTE-1'`, enterpriseID, itemCode).Scan(&count); err != nil || count != 2 {
		t.Fatalf("isolamento por máscara count=%d err=%v", count, err)
	}
}
