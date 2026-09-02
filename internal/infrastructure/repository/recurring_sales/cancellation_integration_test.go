//go:build integration

package recurring_sales_test

import (
	"context"
	"testing"
	"time"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	rsrepo "github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/recurring_sales"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func TestCancellationIsAtomicAuditedAndTenantSafe(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := uuid.New()
	codeA := int64(1_610_000_000 + testutil.UniqueCode()%10_000_000)
	codeB := codeA + 1
	var idA, idB, recurringCode int64
	if _, err := pool.Exec(base, `INSERT INTO users(id,name,email,password) VALUES($1,'Recurring Integration',$2,'x')`, actor, actor.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name) VALUES($1,'Recurring A') RETURNING id`, codeA).Scan(&idA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name) VALUES($1,'Recurring B') RETURNING id`, codeB).Scan(&idB); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO recurring_sales(enterprise_code,customer_code,item_code,movement_type,term_type,sale_date,next_adjustment_date,quantity,unit_value,generated_order_code,created_by) VALUES($1,10,20,'SALE','INDEFINITE',CURRENT_DATE,CURRENT_DATE+INTERVAL '1 year',1,100,999,$2) RETURNING code`, codeA, actor).Scan(&recurringCode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(base, `DELETE FROM recurring_sales_events WHERE enterprise_code=$1`, codeA)
		_, _ = pool.Exec(base, `DELETE FROM recurring_sales WHERE code=$1`, recurringCode)
		_, _ = pool.Exec(base, `DELETE FROM enterprise WHERE id=ANY($1)`, []int64{idA, idB})
		_, _ = pool.Exec(base, `DELETE FROM users WHERE id=$1`, actor)
	})
	ctxA := context.WithValue(base, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), EnterpriseID: idA, EnterpriseCode: codeA})
	ctxB := context.WithValue(base, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), EnterpriseID: idB, EnterpriseCode: codeB})
	repo := recurring_sales.New(pool)
	effective := time.Now().UTC().Truncate(24 * time.Hour)
	result, err := repo.CancelAtomic(ctxA, rsrepo.CancellationCommand{Code: recurringCode, EffectiveDate: effective, FutureOrdersPolicy: "NAO_GERAR_NOVOS", Reason: "encerramento contratual", ActorID: actor, CorrelationID: "integration-cancel"})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsActive || result.LifecycleStatus != "CANCELADA" || result.FutureOrdersPolicy == nil || *result.FutureOrdersPolicy != "NAO_GERAR_NOVOS" {
		t.Fatalf("cancelamento não persistido: %+v", result)
	}
	if _, err = repo.Get(ctxB, recurringCode); err == nil {
		t.Fatal("tenant B consultou recorrência do tenant A")
	}
	var count int
	if err = pool.QueryRow(base, `SELECT count(*) FROM recurring_sales_events WHERE enterprise_code=$1 AND recurring_sale_code=$2 AND actor_id=$3 AND before_state IS NOT NULL AND after_state->>'lifecycle_status'='CANCELADA'`, codeA, recurringCode, actor).Scan(&count); err != nil || count != 1 {
		t.Fatalf("auditoria incompleta: count=%d err=%v", count, err)
	}
}
