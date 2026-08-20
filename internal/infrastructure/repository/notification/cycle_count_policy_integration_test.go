//go:build integration

package notification_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	notificationrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/notification"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/google/uuid"
)

func TestPolicyCycleCountSchedulerLifecycleAndTenantIsolation(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	actor := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users(id,name,email,password) VALUES($1,'Scheduler',$2,'x')`, actor, actor.String()+"@test.local"); err != nil {
		t.Fatal(err)
	}
	var warehouseID, tenantA, tenantB int64
	if err := pool.QueryRow(ctx, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed) VALUES($1,'Scheduler',$2,'NORMAL','INTERNO',TRUE,TRUE) RETURNING id`, fmt.Sprintf("WP-%d", testutil.UniqueCode()), actor).Scan(&warehouseID); err != nil {
		t.Fatal(err)
	}
	base := int64(1_600_000_000 + testutil.UniqueCode()%400_000_000)
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Policy A',$3),($2,'Policy B',$3) RETURNING id`, base, base+1, actor).Scan(&tenantA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM enterprise WHERE code=$1`, base+1).Scan(&tenantB); err != nil {
		t.Fatal(err)
	}
	itemA, itemB := testutil.UniqueCode(), testutil.UniqueCode()
	_, err := pool.Exec(ctx, `INSERT INTO items(warehouse_code,code,name,created_by,enterprise_id,business_code,warehouse_cyclical_count_config) VALUES($1,$2,'Policy A',$4,$5,'0007','{"days":7}'),($1,$3,'Policy B',$4,$6,'0007','{"days":30}')`, warehouseID, itemA, itemB, actor, tenantA, tenantB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE items SET cyclical_count_policy_activated_at=NOW()-INTERVAL '10 days' WHERE enterprise_id=$1 AND code=$2`, tenantA, itemA); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM stock_cycle_count_audit WHERE enterprise_id IN($1,$2)`, tenantA, tenantB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM stock_cycle_counts WHERE enterprise_id IN($1,$2)`, tenantA, tenantB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM items WHERE enterprise_id IN($1,$2)`, tenantA, tenantB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM enterprise WHERE id IN($1,$2)`, tenantA, tenantB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM warehouse WHERE id=$1`, warehouseID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, actor)
	})
	repo := notificationrepo.New(pool)
	if err = repo.SchedulePolicyCycleCounts(ctx); err != nil {
		t.Fatal(err)
	}
	if err = repo.SchedulePolicyCycleCounts(ctx); err != nil { // idempotência
		t.Fatal(err)
	}
	var count int
	var firstID uuid.UUID
	var scheduled time.Time
	if err = pool.QueryRow(ctx, `SELECT id,scheduled_for FROM stock_cycle_counts WHERE enterprise_id=$1 AND item_code=$2 AND origin='POLITICA_ITEM'`, tenantA, itemA).Scan(&firstID, &scheduled); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM stock_cycle_counts WHERE enterprise_id=$1 AND item_code=$2 AND origin='POLITICA_ITEM'`, tenantA, itemA).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 || !scheduled.Before(time.Now()) { // política ativada há 10 dias, intervalo 7: atrasada
		t.Fatalf("primeira programação count=%d scheduled=%s", count, scheduled)
	}
	if err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM stock_cycle_counts WHERE enterprise_id=$1 AND item_code=$2`, tenantB, itemA).Scan(&count); err != nil || count != 0 {
		t.Fatalf("vazamento entre tenants: count=%d err=%v", count, err)
	}
	if err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM stock_cycle_counts WHERE enterprise_id=$1 AND item_code=$2 AND scheduled_for>NOW()`, tenantB, itemB).Scan(&count); err != nil || count != 1 {
		t.Fatalf("primeira programação do outro tenant: count=%d err=%v", count, err)
	}
	approvedAt := time.Now().Add(-48 * time.Hour)
	if _, err = pool.Exec(ctx, `UPDATE stock_cycle_counts SET state='APROVADA',approved_at=$3 WHERE enterprise_id=$1 AND id=$2`, tenantA, firstID, approvedAt); err != nil {
		t.Fatal(err)
	}
	if err = repo.SchedulePolicyCycleCounts(ctx); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT COUNT(*),MAX(scheduled_for) FROM stock_cycle_counts WHERE enterprise_id=$1 AND item_code=$2`, tenantA, itemA).Scan(&count, &scheduled); err != nil {
		t.Fatal(err)
	}
	if count != 2 || scheduled.Before(approvedAt.Add(7*24*time.Hour-time.Second)) || scheduled.After(approvedAt.Add(7*24*time.Hour+time.Second)) {
		t.Fatalf("recorrência incorreta count=%d scheduled=%s", count, scheduled)
	}
	if _, err = pool.Exec(ctx, `UPDATE items SET warehouse_cyclical_count_config=NULL WHERE enterprise_id=$1 AND code=$2`, tenantA, itemA); err != nil {
		t.Fatal(err)
	}
	if err = repo.SchedulePolicyCycleCounts(ctx); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM stock_cycle_counts WHERE enterprise_id=$1 AND item_code=$2 AND state='PROGRAMADA'`, tenantA, itemA).Scan(&count); err != nil || count != 0 {
		t.Fatalf("desativação manteve programação futura: count=%d err=%v", count, err)
	}
	if err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM stock_cycle_counts WHERE enterprise_id=$1 AND item_code=$2 AND state='APROVADA'`, tenantA, itemA).Scan(&count); err != nil || count != 1 {
		t.Fatalf("desativação alterou histórico aprovado: count=%d err=%v", count, err)
	}
}
