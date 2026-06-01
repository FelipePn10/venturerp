//go:build integration

// Package testutil provides shared helpers for integration tests that need a
// real Postgres. These tests are compiled only with the `integration` build tag
// and skipped unless TEST_DATABASE_URL points at a migrated database.
//
//	make test-integration            # uses TEST_DATABASE_URL
//	TEST_DATABASE_URL=... go test -tags=integration ./...
package testutil

import (
	"context"
	"math/rand"
	"os"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool connects to TEST_DATABASE_URL or skips the test when it is unset.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("integration: set TEST_DATABASE_URL to a migrated Postgres to run")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connecting to TEST_DATABASE_URL: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatalf("pinging TEST_DATABASE_URL: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// Queries returns a sqlc.Queries bound to the test pool, plus the pool itself.
func Queries(t *testing.T) (*sqlc.Queries, *pgxpool.Pool) {
	p := Pool(t)
	return sqlc.New(p), p
}

// UniqueCode returns a large code in a band unlikely to collide with seeded data,
// so integration tests can create and clean up their own rows safely.
func UniqueCode() int64 {
	return 9_000_000_000 + rand.Int63n(900_000_000)
}

// Exec runs a statement (used for deterministic cleanup in defers).
func Exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Logf("cleanup exec failed (%s): %v", sql, err)
	}
}
