//go:build integration

package user

import (
	"context"
	"os"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIdentityMirrorUsesProductionAuthorityAndIsIdempotent(t *testing.T) {
	authorityURL := os.Getenv("IDENTITY_TEST_DATABASE_URL")
	targetURL := os.Getenv("TRAINING_TEST_DATABASE_URL")
	if authorityURL == "" || targetURL == "" {
		t.Skip("set IDENTITY_TEST_DATABASE_URL and TRAINING_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	authority, err := pgxpool.New(ctx, authorityURL)
	if err != nil {
		t.Fatal(err)
	}
	defer authority.Close()
	target, err := pgxpool.New(ctx, targetURL)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()

	userID := uuid.New()
	const enterpriseID int64 = 910000001
	const enterpriseCode int64 = 910001
	email := userID.String() + "@identity.test"
	cleanup := func(pool *pgxpool.Pool) {
		_, _ = pool.Exec(ctx, `DELETE FROM user_enterprises WHERE user_id=$1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM enterprise WHERE id=$1`, enterpriseID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
	}
	cleanup(authority)
	cleanup(target)
	t.Cleanup(func() { cleanup(target); cleanup(authority) })

	if _, err = authority.Exec(ctx, `INSERT INTO users(id,name,email,password,role,auth_version,is_active)
		VALUES($1,'Treinando',$2,'production-password-hash','USER',1,true)`, userID, email); err != nil {
		t.Fatal(err)
	}
	if _, err = authority.Exec(ctx, `INSERT INTO enterprise(id,code,name) VALUES($1,$2,'Empresa de teste')`, enterpriseID, enterpriseCode); err != nil {
		t.Fatal(err)
	}
	if _, err = authority.Exec(ctx, `INSERT INTO user_enterprises(user_id,enterprise_id,role) VALUES($1,$2,'USER')`, userID, enterpriseID); err != nil {
		t.Fatal(err)
	}

	mirror := NewIdentityMirror(authority, target)
	for range 2 {
		if err = mirror.SyncIdentity(ctx, userID.String(), enterpriseID); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	var mirroredPassword, mirroredRole string
	var mirroredVersion int64
	if err = target.QueryRow(ctx, `SELECT COUNT(*),MIN(password),MIN(role),MIN(auth_version)
		FROM users WHERE id=$1`, userID).Scan(&count, &mirroredPassword, &mirroredRole, &mirroredVersion); err != nil {
		t.Fatal(err)
	}
	if count != 1 || mirroredPassword != "!managed-by-production-identity-authority" || mirroredRole != "USER" || mirroredVersion != 1 {
		t.Fatalf("unexpected first mirror: count=%d password=%q role=%q version=%d", count, mirroredPassword, mirroredRole, mirroredVersion)
	}

	if _, err = authority.Exec(ctx, `UPDATE users SET role='ADMIN',auth_version=2,is_active=false WHERE id=$1`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err = authority.Exec(ctx, `UPDATE user_enterprises SET role='ADMIN' WHERE user_id=$1 AND enterprise_id=$2`, userID, enterpriseID); err != nil {
		t.Fatal(err)
	}
	if err = mirror.SyncIdentity(ctx, userID.String(), enterpriseID); err != nil {
		t.Fatal(err)
	}
	var active bool
	if err = target.QueryRow(ctx, `SELECT role,auth_version,is_active FROM users WHERE id=$1`, userID).Scan(&mirroredRole, &mirroredVersion, &active); err != nil {
		t.Fatal(err)
	}
	if mirroredRole != "ADMIN" || mirroredVersion != 2 || active {
		t.Fatalf("authority update not mirrored: role=%q version=%d active=%v", mirroredRole, mirroredVersion, active)
	}

	repo := NewRepositoryUserSQLC(sqlc.New(authority), authority)
	if _, err = repo.FindByEmail(ctx, email); err == nil {
		t.Fatal("inactive authoritative user was allowed to authenticate")
	}
	if _, err = repo.CurrentAuthorization(ctx, userID.String(), enterpriseID); err == nil {
		t.Fatal("inactive authoritative user kept a valid JWT authorization")
	}
}
