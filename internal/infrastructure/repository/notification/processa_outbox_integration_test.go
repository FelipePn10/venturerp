//go:build integration

package notification_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	notificationentity "github.com/FelipePn10/panossoerp/internal/domain/notification/entity"
	notificationrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/notification"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Com a assinatura ligada, o evento vira alerta e entrega. A instrução tinha tipos
// inconsistentes nos parâmetros: a preparação falhava e nenhum alerta saía. Em
// produção ninguém viu porque todas as assinaturas estavam desligadas.
func TestProcessarOutboxComAssinaturaLigada(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	actor := testutil.Actor(t, pool)
	code := int64(1_700_000_000 + testutil.UniqueCode()%200_000_000)
	var empresa int64
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Outbox',$2) RETURNING id`, code, actor).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM notification_deliveries WHERE enterprise_id=$1`, `DELETE FROM notification_alerts WHERE enterprise_id=$1`,
			`DELETE FROM notification_outbox WHERE enterprise_id=$1`, `DELETE FROM enterprise_notification_recipients WHERE enterprise_id=$1`,
			`DELETE FROM enterprise_notification_subscriptions WHERE enterprise_id=$1`, `DELETE FROM enterprise_notification_settings WHERE enterprise_id=$1`,
			`DELETE FROM user_enterprises WHERE enterprise_id=$1`, `DELETE FROM enterprise WHERE id=$1`,
		} {
			_, _ = pool.Exec(context.Background(), q, empresa)
		}
	})
	var evento string
	var versao int
	if err := pool.QueryRow(ctx, `SELECT event_key, version FROM notification_event_catalog WHERE event_key='ESTOQUE_ABAIXO_MINIMO' ORDER BY version DESC LIMIT 1`).Scan(&evento, &versao); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO enterprise_notification_settings(enterprise_id,enabled) VALUES($1,TRUE)
		ON CONFLICT (enterprise_id) DO UPDATE SET enabled=TRUE`, empresa)
	var assinatura uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise_notification_subscriptions(enterprise_id,event_key,event_version,enabled,cadence,created_by)
		VALUES($1,$2,$3,TRUE,'IMEDIATO',$4)
		ON CONFLICT (enterprise_id,event_key) DO UPDATE SET enabled=TRUE, cadence='IMEDIATO' RETURNING id`, empresa, evento, versao, actor).Scan(&assinatura); err != nil {
		t.Fatal(err)
	}
	destinatario := uuid.New()
	testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password,role,is_active,auth_version) VALUES($1,'Destino',$2,'x','ADMIN',TRUE,1)`, destinatario, destinatario.String()+"@test.local")
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, destinatario) })
	testutil.Exec(t, pool, `INSERT INTO user_enterprises(user_id,enterprise_id,role) VALUES($1,$2,'ADMIN')`, destinatario, empresa)
	testutil.Exec(t, pool, `INSERT INTO enterprise_notification_recipients(enterprise_id,subscription_id,recipient_type,user_id) VALUES($1,$2,'USUARIO',$3)`, empresa, assinatura, destinatario)
	var id uuid.UUID
	carga := json.RawMessage(`{"descricao":"Abaixo do mínimo"}`)
	if err := pool.QueryRow(ctx, `INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key)
		VALUES($1,$2,$3,'SALDO_ESTOQUE','1','X',$4,'k1') RETURNING id`, empresa, evento, versao, carga).Scan(&id); err != nil {
		t.Fatal(err)
	}
	repo := notificationrepo.New(pool)
	err := repo.ProcessOutbox(ctx, notificationentity.OutboxEvent{ID: id, EnterpriseID: empresa, EventKey: evento, EventVersion: versao,
		AggregateType: "SALDO_ESTOQUE", AggregateInternalID: "1", AggregatePublicID: "X", Payload: carga, DeduplicationKey: "k1", OccurredAt: time.Now()})
	if err != nil {
		t.Fatalf("processar o evento: %v", err)
	}
	var alertas, entregas int
	if err := pool.QueryRow(ctx, `SELECT (SELECT COUNT(*) FROM notification_alerts WHERE enterprise_id=$1), (SELECT COUNT(*) FROM notification_deliveries WHERE enterprise_id=$1)`, empresa).Scan(&alertas, &entregas); err != nil {
		t.Fatal(err)
	}
	if alertas != 1 || entregas != 1 {
		t.Fatalf("alertas %d, entregas %d", alertas, entregas)
	}
}
