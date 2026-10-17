//go:build integration

package notification_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	notificationentity "github.com/FelipePn10/panossoerp/internal/domain/notification/entity"
	notificationrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/notification"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A contagem divergente aprovada ajusta o saldo — uma vez, na aprovação.
func TestCycleCountAprovadaAjustaOSaldo(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	actor := testutil.Actor(t, pool)
	code := int64(1_700_000_000 + testutil.UniqueCode()%200_000_000)
	var empresa, almox int64
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Contagem ajuste',$2) RETURNING id`, code, actor).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,'Contagem',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("WA-%d", code), actor, empresa).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	item := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO items(warehouse_code,code,name,created_by,enterprise_id,business_code) VALUES($1,$2,'Item',$3,$4,'AJ-01')`, almox, item, actor, empresa)
	testutil.Exec(t, pool, `INSERT INTO stock_balances(enterprise_id,item_code,warehouse_id,mask,quantity,avg_cost,total_cost) VALUES($1,$2,$3,'',5,4,20)`, empresa, item, almox)
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM notification_outbox WHERE enterprise_id=$1`, `DELETE FROM notification_alerts WHERE enterprise_id=$1`,
			`DELETE FROM stock_cycle_count_audit WHERE enterprise_id=$1`, `DELETE FROM stock_cycle_counts WHERE enterprise_id=$1`,
			`DELETE FROM stock_movements WHERE enterprise_id=$1`, `DELETE FROM stock_balances WHERE enterprise_id=$1`,
			`DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`, `DELETE FROM enterprise WHERE id=$1`,
		} {
			_, _ = pool.Exec(context.Background(), q, empresa)
		}
	})
	var id uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO stock_cycle_counts(enterprise_id,warehouse_id,item_code,scheduled_for,origin) VALUES($1,$2,$3,$4,'MANUAL') RETURNING id`,
		empresa, almox, item, time.Now()).Scan(&id); err != nil {
		t.Fatal(err)
	}
	repo := notificationrepo.New(pool)
	saldo := func() float64 {
		var q float64
		_ = pool.QueryRow(ctx, `SELECT SUM(quantity)::float8 FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, empresa, item).Scan(&q)
		return q
	}
	if _, err := repo.TransitionCycleCount(ctx, empresa, id, actor, notificationentity.CycleCounting, nil, "início"); err != nil {
		t.Fatal(err)
	}
	contado := "3"
	c, err := repo.TransitionCycleCount(ctx, empresa, id, actor, notificationentity.CycleCompleted, &contado, "contei 3")
	if err != nil || c.State != notificationentity.CycleDivergent {
		t.Fatalf("5 esperado, 3 contado: divergente. %s %v", c.State, err)
	}
	if q := saldo(); q != 5 {
		t.Fatalf("divergente ainda não ajusta: %v", q)
	}
	if _, err := repo.TransitionCycleCount(ctx, empresa, id, actor, notificationentity.CycleCompleted, nil, "divergência aceita"); err != nil {
		t.Fatal(err)
	}
	if q := saldo(); q != 5 {
		t.Fatalf("concluída depois de divergente espera a aprovação: %v", q)
	}
	if _, err := repo.TransitionCycleCount(ctx, empresa, id, actor, notificationentity.CycleApproved, nil, "aprovado"); err != nil {
		t.Fatal(err)
	}
	if q := saldo(); q != 3 {
		t.Fatalf("aprovada: saldo ajustado para 3, veio %v", q)
	}
	var movs int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM stock_movements WHERE enterprise_id=$1 AND reference_type='CONTAGEM_CICLICA' AND quantity=-2`, empresa).Scan(&movs); err != nil || movs != 1 {
		t.Fatalf("um ajuste de −2 referenciando a contagem: %d %v", movs, err)
	}
}
