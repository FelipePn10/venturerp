//go:build integration

package testutil

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
)

// TenantContext mirrors both enterprise keys carried by authenticated requests.
func TenantContext(t *testing.T, pool *pgxpool.Pool) context.Context {
	t.Helper()
	ctx := context.Background()
	u := &security.AuthUser{Role: "ADMIN"}
	if err := pool.QueryRow(ctx, "SELECT id,code FROM enterprise ORDER BY id LIMIT 1").Scan(&u.EnterpriseID, &u.EnterpriseCode); err != nil {
		t.Fatal(err)
	}
	u.ID = Actor(t, pool).String()
	return context.WithValue(ctx, contextkey.UserKey, u)
}
func Actor(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT id FROM users ORDER BY id LIMIT 1").Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
func EnterpriseID(t *testing.T, ctx context.Context) int64 {
	t.Helper()
	id, err := tenant.ID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
func SeedItem(t *testing.T, pool *pgxpool.Pool, ctx context.Context, code int64, actor uuid.UUID) {
	t.Helper()
	if _, err := pool.Exec(ctx, "INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,$1,$2,$3)", code, actor, EnterpriseID(t, ctx)); err != nil {
		t.Fatal(err)
	}
}
