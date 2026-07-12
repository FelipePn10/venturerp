//go:build integration

package mrp_calculation

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func TestSuggestionNumberAndItemLLCArePersisted(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	userID := uuid.New()
	const enterpriseCode, planCode, orderNumber, itemBusinessCode = 930001, 939001, 938001, 937001
	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM mrp_planned_suggestions WHERE plan_code=$1`, planCode)
		_, _ = pool.Exec(ctx, `DELETE FROM production_plans WHERE code=$1`, planCode)
		_, _ = pool.Exec(ctx, `DELETE FROM items WHERE code=$1`, itemBusinessCode)
		_, _ = pool.Exec(ctx, `DELETE FROM enterprise WHERE code=$1`, enterpriseCode)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email='mrp-llc-930001@test.local'`)
	}
	cleanup()
	t.Cleanup(cleanup)

	testutil.Exec(t, pool, `INSERT INTO users (id,name,email,password) VALUES ($1,'MRP LLC','mrp-llc-930001@test.local','x')`, userID)
	testutil.Exec(t, pool, `INSERT INTO enterprise (code,name,created_by) VALUES ($1,'MRP LLC enterprise',$2)`, enterpriseCode, userID)
	var enterpriseID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM enterprise WHERE code=$1`, enterpriseCode).Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO production_plans (code,name,created_by,enterprise_id) VALUES ($1,'LLC plan',$2,$3)`, planCode, userID, enterpriseID)
	var itemID int64
	if err := pool.QueryRow(ctx, `INSERT INTO items (warehouse_code,code,health,created_by) VALUES (0,$1,'ATIVO',$2) RETURNING id`, itemBusinessCode, userID).Scan(&itemID); err != nil {
		t.Fatal(err)
	}

	repo := NewMRPCalculationRepositorySQLC(sqlc.New(pool), pool)
	tenantCtx := context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	number := int64(orderNumber)
	warehouseCode := int64(936001)
	sourceEnterpriseCode := int64(935001)
	created, err := repo.CreatePlannedOrderSuggestion(tenantCtx, &entity.PlannedOrderSuggestion{
		OrderNumber: &number, PlanCode: planCode, ItemCode: itemID, Quantity: 2,
		NeedDate: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), OrderType: "TECHNICAL_ASSISTANCE", DemandType: "INTER_FACTORY", LLC: 3, WarehouseCode: &warehouseCode,
		InterFactory: true, SourceEnterpriseCode: &sourceEnterpriseCode, AutoRelease: true,
	})
	if err != nil || created.OrderNumber == nil || *created.OrderNumber != orderNumber || created.WarehouseCode == nil || *created.WarehouseCode != warehouseCode || !created.InterFactory || created.SourceEnterpriseCode == nil || *created.SourceEnterpriseCode != sourceEnterpriseCode || !created.AutoRelease {
		t.Fatalf("unexpected numbered suggestion: %#v, %v", created, err)
	}
	if _, err := repo.CreatePlannedOrderSuggestion(tenantCtx, &entity.PlannedOrderSuggestion{
		OrderNumber: &number, PlanCode: planCode, ItemCode: itemID, Quantity: 1,
		NeedDate: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), OrderType: "FABRICACAO", DemandType: "INDEPENDENTE",
	}); err == nil {
		t.Fatal("expected duplicated suggestion order number to be rejected")
	}
	if err := repo.UpdateItemLLCs(tenantCtx, map[int64]int{itemID: 4}); err != nil {
		t.Fatal(err)
	}
	var llc int
	if err := pool.QueryRow(ctx, `SELECT planning_llc FROM items WHERE id=$1`, itemID).Scan(&llc); err != nil || llc != 4 {
		t.Fatalf("unexpected persisted LLC %d, %v", llc, err)
	}
}
