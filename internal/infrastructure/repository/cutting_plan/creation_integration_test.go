//go:build integration

package cutting_plan

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func TestCreatePlanUsesAuthenticatedTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := uuid.New()
	code := int64(1_000_000_000 + testutil.UniqueCode()%500_000_000)
	var enterpriseID int64
	testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password) VALUES($1,'Plano de corte',$2,'x')`, actor, actor.String()+"@test.local")
	if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Plano de corte',$2) RETURNING id`, code, actor).Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(base, `DELETE FROM cutting_plans WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(base, `DELETE FROM enterprise WHERE id=$1`, enterpriseID)
		_, _ = pool.Exec(base, `DELETE FROM users WHERE id=$1`, actor)
	})
	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	repo := New(sqlc.New(pool), pool)
	planCode, err := repo.NextPlanCode(ctx)
	if err != nil {
		t.Fatal(err)
	}
	created, err := repo.CreatePlan(ctx, &entity.CuttingPlan{Code: planCode, CutType: entity.CutTypeLinear1D, Source: entity.SourceManual, MaterialItemCode: testutil.UniqueCode(), StockUoM: types.UN, UoMFactor: 1, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	var persistedTenant int64
	if err = pool.QueryRow(base, `SELECT enterprise_id FROM cutting_plans WHERE id=$1`, created.ID).Scan(&persistedTenant); err != nil {
		t.Fatal(err)
	}
	if persistedTenant != enterpriseID {
		t.Fatalf("enterprise_id=%d, esperado=%d", persistedTenant, enterpriseID)
	}
	part, err := repo.AddPart(ctx, &entity.CuttingPlanPart{PlanID: created.ID, Label: "Peça", LengthMM: 100, Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(base, `SELECT enterprise_id FROM cutting_plan_parts WHERE id=$1`, part.ID).Scan(&persistedTenant); err != nil {
		t.Fatal(err)
	}
	if persistedTenant != enterpriseID {
		t.Fatalf("enterprise_id da peça=%d, esperado=%d", persistedTenant, enterpriseID)
	}
}
