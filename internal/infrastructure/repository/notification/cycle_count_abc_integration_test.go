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

// Item só com classe ABC é contado pela política da classe, a contagem não é
// cancelada na rodada seguinte, e a contagem vencida vira alerta quando a
// empresa liga o alerta — também a que já estava vencida antes de ligar.
func TestContagemPelaClasseABCEAlertaDeVencida(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	ator := testutil.Actor(t, pool)
	code := int64(1_800_000_000 + testutil.UniqueCode()%100_000_000)
	var empresa int64
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Contagem ABC',$2) RETURNING id`, code, ator).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	almox := testutil.NovoAlmoxarifado(t, pool, empresa)
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM notification_deliveries WHERE enterprise_id=$1`, `DELETE FROM notification_alerts WHERE enterprise_id=$1`,
			`DELETE FROM notification_outbox WHERE enterprise_id=$1`, `DELETE FROM enterprise_notification_recipients WHERE enterprise_id=$1`,
			`DELETE FROM enterprise_notification_subscriptions WHERE enterprise_id=$1`, `DELETE FROM enterprise_notification_settings WHERE enterprise_id=$1`,
			`DELETE FROM stock_cycle_count_audit WHERE enterprise_id=$1`, `DELETE FROM stock_cycle_counts WHERE enterprise_id=$1`,
			`DELETE FROM stock_abc_count_policy WHERE enterprise_id=$1`, `DELETE FROM items WHERE enterprise_id=$1`,
			`DELETE FROM warehouse WHERE enterprise_id=$1`, `DELETE FROM user_enterprises WHERE enterprise_id=$1`, `DELETE FROM enterprise WHERE id=$1`,
		} {
			_, _ = pool.Exec(context.Background(), q, empresa)
		}
	})
	// Política A de 90 dias, que começou há 100 dias: o item A, cadastrado antes, está 10 dias atrasado.
	testutil.Exec(t, pool, `INSERT INTO stock_abc_count_policy(enterprise_id,abc_class,days_interval,created_at,updated_at)
		VALUES($1,'A',90,NOW()-INTERVAL '100 days',NOW()-INTERVAL '100 days')
		ON CONFLICT (enterprise_id,abc_class) DO UPDATE SET days_interval=90, created_at=EXCLUDED.created_at`, empresa)
	ativo, inativo := testutil.UniqueCode(), testutil.UniqueCode()
	for _, it := range []struct {
		codigo int64
		saude  string
	}{{ativo, "ATIVO"}, {inativo, "INATIVO"}} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,planning_abc_class,health,created_at)
			VALUES($1,$2,$3,$4,$5,'A',$6,NOW()-INTERVAL '200 days')`, it.codigo, fmt.Sprint(it.codigo), almox, ator, empresa, it.saude)
	}
	repo := notificationrepo.New(pool)
	for i := 0; i < 2; i++ { // a segunda rodada não pode cancelar a primeira
		if err := repo.SchedulePolicyCycleCounts(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var programadas, canceladas, doInativo int
	var programadaPara time.Time
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FILTER (WHERE state='PROGRAMADA' AND item_code=$2), COUNT(*) FILTER (WHERE state='CANCELADA'),
			COUNT(*) FILTER (WHERE item_code=$3), COALESCE(MAX(scheduled_for) FILTER (WHERE item_code=$2), NOW())
		  FROM stock_cycle_counts WHERE enterprise_id=$1`, empresa, ativo, inativo).Scan(&programadas, &canceladas, &doInativo, &programadaPara); err != nil {
		t.Fatal(err)
	}
	if programadas != 1 || canceladas != 0 || doInativo != 0 {
		t.Fatalf("programadas %d, canceladas %d, do inativo %d", programadas, canceladas, doInativo)
	}
	if atraso := time.Since(programadaPara); atraso < 9*24*time.Hour || atraso > 11*24*time.Hour {
		t.Fatalf("a primeira contagem vence no início da política + 90 dias (há 10 dias): %s", programadaPara)
	}

	eventos := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM notification_outbox WHERE enterprise_id=$1 AND event_key='ESTOQUE_CONTAGEM_VENCIDA'`, empresa).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if err := repo.ScheduleOperationalAlerts(ctx); err != nil {
		t.Fatal(err)
	}
	if n := eventos(); n != 0 {
		t.Fatalf("alerta desligado gerou %d eventos", n)
	}

	// A empresa liga o alerta em VNOT0100, com um destinatário.
	testutil.Exec(t, pool, `INSERT INTO enterprise_notification_settings(enterprise_id,enabled) VALUES($1,TRUE) ON CONFLICT (enterprise_id) DO UPDATE SET enabled=TRUE`, empresa)
	var versao int
	if err := pool.QueryRow(ctx, `SELECT version FROM notification_event_catalog WHERE event_key='ESTOQUE_CONTAGEM_VENCIDA' ORDER BY version DESC LIMIT 1`).Scan(&versao); err != nil {
		t.Fatal(err)
	}
	var assinatura uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise_notification_subscriptions(enterprise_id,event_key,event_version,enabled,cadence,created_by)
		VALUES($1,'ESTOQUE_CONTAGEM_VENCIDA',$2,TRUE,'IMEDIATO',$3) RETURNING id`, empresa, versao, ator).Scan(&assinatura); err != nil {
		t.Fatal(err)
	}
	destino := uuid.New()
	testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password,role,is_active,auth_version) VALUES($1,'Almoxarife',$2,'x','ADMIN',TRUE,1)`, destino, destino.String()+"@test.local")
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, destino) })
	testutil.Exec(t, pool, `INSERT INTO user_enterprises(user_id,enterprise_id,role) VALUES($1,$2,'ADMIN')`, destino, empresa)
	testutil.Exec(t, pool, `INSERT INTO enterprise_notification_recipients(enterprise_id,subscription_id,recipient_type,user_id) VALUES($1,$2,'USUARIO',$3)`, empresa, assinatura, destino)

	if err := repo.ScheduleOperationalAlerts(ctx); err != nil {
		t.Fatal(err)
	}
	if n := eventos(); n != 1 {
		t.Fatalf("com o alerta ligado, a contagem já vencida gera 1 evento: %d", n)
	}
	var e notificationentity.OutboxEvent
	if err := pool.QueryRow(ctx, `SELECT id,enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key,occurred_at
		  FROM notification_outbox WHERE enterprise_id=$1 AND event_key='ESTOQUE_CONTAGEM_VENCIDA'`, empresa).
		Scan(&e.ID, &e.EnterpriseID, &e.EventKey, &e.EventVersion, &e.AggregateType, &e.AggregateInternalID, &e.AggregatePublicID, &e.Payload, &e.DeduplicationKey, &e.OccurredAt); err != nil {
		t.Fatal(err)
	}
	if err := repo.ProcessOutbox(ctx, e); err != nil {
		t.Fatal(err)
	}
	var alertas, entregas int
	if err := pool.QueryRow(ctx, `SELECT (SELECT COUNT(*) FROM notification_alerts WHERE enterprise_id=$1 AND event_key='ESTOQUE_CONTAGEM_VENCIDA' AND state='ABERTO'),
			(SELECT COUNT(*) FROM notification_deliveries WHERE enterprise_id=$1)`, empresa).Scan(&alertas, &entregas); err != nil {
		t.Fatal(err)
	}
	if alertas != 1 || entregas != 1 {
		t.Fatalf("a contagem vencida vira alerta aberto e entrega: alertas %d, entregas %d", alertas, entregas)
	}
}
