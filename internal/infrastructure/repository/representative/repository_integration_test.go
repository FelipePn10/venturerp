//go:build integration

package representative_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	representativepg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/representative"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func representativeTenantContext(enterpriseID int64) context.Context {
	return context.WithValue(context.Background(), contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
}

func TestBlockUnblockAndGetAreIsolatedByEnterprise(t *testing.T) {
	pool := testutil.Pool(t)
	repo := representativepg.New(pool)
	tenantA, tenantB := testutil.UniqueCode(), testutil.UniqueCode()
	var representativeA, representativeB int64
	if err := pool.QueryRow(context.Background(), `INSERT INTO representatives(name,document_number) VALUES('Representante A',$1) RETURNING code`, "REP-A").Scan(&representativeA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `INSERT INTO representatives(name,document_number) VALUES('Representante B',$1) RETURNING code`, "REP-B").Scan(&representativeB); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM representative_enterprises WHERE representative_code=ANY($1)`, []int64{representativeA, representativeB})
		_, _ = pool.Exec(context.Background(), `DELETE FROM representatives WHERE code=ANY($1)`, []int64{representativeA, representativeB})
	})
	if _, err := pool.Exec(context.Background(), `INSERT INTO representative_enterprises(representative_code,enterprise_code) VALUES($1,$2),($3,$4)`, representativeA, tenantA, representativeB, tenantB); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.Get(representativeTenantContext(tenantA), representativeB); err == nil {
		t.Fatal("tenant A consultou representante exclusivo do tenant B")
	}
	if err := repo.Block(representativeTenantContext(tenantA), representativeB, "forjado"); err == nil {
		t.Fatal("tenant A bloqueou representante exclusivo do tenant B")
	}
	if err := repo.Block(representativeTenantContext(tenantA), representativeA, "teste"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Unblock(representativeTenantContext(tenantA), representativeA); err != nil {
		t.Fatal(err)
	}
	var blocked bool
	if err := pool.QueryRow(context.Background(), `SELECT blocked FROM representatives WHERE code=$1`, representativeA).Scan(&blocked); err != nil || blocked {
		t.Fatalf("desbloqueio não persistido: blocked=%v err=%v", blocked, err)
	}
}
