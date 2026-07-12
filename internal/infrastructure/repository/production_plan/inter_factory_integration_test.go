//go:build integration

package production_plan

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	planentity "github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func TestInterFactoryConfigurationIsTenantIsolated(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	userID := uuid.New()
	const ownerCode, otherOwnerCode, sourceCode, planCode = 910001, 910002, 910003, 919001
	testutil.Exec(t, pool, `INSERT INTO users (id,name,email,password) VALUES ($1,'MRP tenant','mrp-tenant-910001@test.local','x')`, userID)
	testutil.Exec(t, pool, `INSERT INTO enterprise (code,name,created_by) VALUES ($1,'Owner',$4),($2,'Other owner',$4),($3,'Source',$4)`, ownerCode, otherOwnerCode, sourceCode, userID)
	var ownerID, otherOwnerID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM enterprise WHERE code=$1`, ownerCode).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM enterprise WHERE code=$1`, otherOwnerCode).Scan(&otherOwnerID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO production_plans (code,name,created_by,enterprise_id) VALUES ($1,'Plan',$2,$3)`, planCode, userID, ownerID)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM production_plans WHERE code=$1`, planCode)
		_, _ = pool.Exec(ctx, `DELETE FROM enterprise WHERE code IN ($1,$2,$3)`, ownerCode, otherOwnerCode, sourceCode)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
	})

	repo := NewProductionPlanRepositorySQLC(sqlc.New(pool))
	ownerCtx := context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: ownerID})
	items, err := repo.ReplaceInterFactories(ownerCtx, planCode, []*planentity.InterFactoryEnterprise{{EnterpriseCode: sourceCode, AutoRelease: true}})
	if err != nil || len(items) != 1 || !items[0].AutoRelease {
		t.Fatalf("unexpected replace result: %#v, %v", items, err)
	}

	otherCtx := context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: otherOwnerID})
	items, err = repo.ListInterFactories(otherCtx, planCode)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("other tenant accessed configuration: %#v", items)
	}
}
