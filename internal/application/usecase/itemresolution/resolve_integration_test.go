//go:build integration

package itemresolution_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	itempg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func TestResolveBusinessCodeIsIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	var enterpriseA, enterpriseB int64
	if err = tx.QueryRow(context.Background(), `SELECT id FROM enterprise ORDER BY id LIMIT 1`).Scan(&enterpriseA); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(context.Background(), `SELECT id FROM enterprise WHERE id<>$1 ORDER BY id LIMIT 1`, enterpriseA).Scan(&enterpriseB); err != nil {
		t.Skip("o banco de integração precisa de duas empresas")
	}
	base := testutil.UniqueCode()
	businessCode := fmt.Sprintf("TENANT-%d", base)
	var actor string
	if err = tx.QueryRow(context.Background(), `SELECT id::text FROM users ORDER BY created_at LIMIT 1`).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	for i, enterpriseID := range []int64{enterpriseA, enterpriseB} {
		_, err = tx.Exec(context.Background(), `INSERT INTO items(enterprise_id,business_code,warehouse_code,code,name,nature,engineering_weight,created_by) VALUES($1,$2,1,$3,'Item tenant',2,'{"gross":1,"net":1,"unit":"KG"}'::jsonb,$4::uuid)`, enterpriseID, businessCode, base+int64(i), actor)
		if err != nil {
			t.Fatal(err)
		}
	}
	repo := itempg.NewRepositoryItemSQLC(sqlc.New(tx))
	ctxA := context.WithValue(context.Background(), contextkey.UserKey, &appsecurity.AuthUser{EnterpriseID: enterpriseA})
	ctxB := context.WithValue(context.Background(), contextkey.UserKey, &appsecurity.AuthUser{EnterpriseID: enterpriseB})
	itemA, err := itemresolution.Resolve(ctxA, repo, request.TextCode(businessCode))
	if err != nil {
		t.Fatal(err)
	}
	itemB, err := itemresolution.Resolve(ctxB, repo, request.TextCode(businessCode))
	if err != nil {
		t.Fatal(err)
	}
	if itemA.Code == itemB.Code || int64(itemA.Code) != base || int64(itemB.Code) != base+1 {
		t.Fatalf("resolução cruzou tenants: A=%d B=%d", itemA.Code, itemB.Code)
	}
}
