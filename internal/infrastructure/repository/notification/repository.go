package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	notificationentity "github.com/FelipePn10/panossoerp/internal/domain/notification/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// PrometheusMetrics returns only bounded labels (catalog module/severity and
// persisted states); tenant IDs, event keys, recipients and payload data are
// deliberately excluded.
func (r *Repository) PrometheusMetrics(ctx context.Context) (string, error) {
	var b strings.Builder
	var outboxSize int64
	var oldest float64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*),COALESCE(EXTRACT(EPOCH FROM NOW()-MIN(occurred_at)),0) FROM notification_outbox WHERE state IN('PENDENTE','PROCESSANDO')`).Scan(&outboxSize, &oldest); err != nil {
		return "", err
	}
	b.WriteString("# TYPE notification_outbox_size gauge\n# TYPE notification_outbox_oldest_age_seconds gauge\n")
	fmt.Fprintf(&b, "notification_outbox_size %d\nnotification_outbox_oldest_age_seconds %g\n", outboxSize, oldest)
	rows, err := r.pool.Query(ctx, `SELECT c.module,a.severity,COUNT(*) FROM notification_alerts a JOIN notification_event_catalog c ON c.event_key=a.event_key AND c.version=a.event_version WHERE a.state='ABERTO' GROUP BY c.module,a.severity ORDER BY c.module,a.severity`)
	if err != nil {
		return "", err
	}
	b.WriteString("# TYPE notification_open_alerts gauge\n")
	for rows.Next() {
		var module, severity string
		var count int64
		if err = rows.Scan(&module, &severity, &count); err != nil {
			rows.Close()
			return "", err
		}
		fmt.Fprintf(&b, "notification_open_alerts{module=%s,severity=%s} %d\n", strconv.Quote(module), strconv.Quote(severity), count)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()
	rows, err = r.pool.Query(ctx, `SELECT state,COUNT(*) FROM notification_deliveries GROUP BY state ORDER BY state`)
	if err != nil {
		return "", err
	}
	b.WriteString("# TYPE notification_deliveries gauge\n")
	for rows.Next() {
		var state string
		var count int64
		if err = rows.Scan(&state, &count); err != nil {
			rows.Close()
			return "", err
		}
		fmt.Fprintf(&b, "notification_deliveries{state=%s} %d\n", strconv.Quote(state), count)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()
	var deadLetters, attempts, digests, throttled int64
	var latency float64
	err = r.pool.QueryRow(ctx, `SELECT (SELECT COUNT(*) FROM notification_dead_letters WHERE retried_at IS NULL),(SELECT COUNT(*) FROM notification_delivery_attempts),(SELECT COUNT(*) FROM notification_digest_runs WHERE state='ENVIADO'),(SELECT COALESCE(SUM(message_count),0) FROM notification_provider_rate_windows WHERE window_start>NOW()-INTERVAL '2 minutes'),(SELECT COALESCE(AVG(EXTRACT(EPOCH FROM sent_at-created_at)),0) FROM notification_deliveries WHERE sent_at IS NOT NULL)`).Scan(&deadLetters, &attempts, &digests, &throttled, &latency)
	if err != nil {
		return "", err
	}
	b.WriteString("# TYPE notification_dead_letters gauge\n# TYPE notification_delivery_attempts counter\n# TYPE notification_digests_sent counter\n# TYPE notification_provider_recent_messages gauge\n# TYPE notification_event_to_delivery_seconds gauge\n")
	fmt.Fprintf(&b, "notification_dead_letters %d\nnotification_delivery_attempts %d\nnotification_digests_sent %d\nnotification_provider_recent_messages %d\nnotification_event_to_delivery_seconds %g\n", deadLetters, attempts, digests, throttled, latency)
	return b.String(), nil
}

func (r *Repository) Enqueue(ctx context.Context, e notificationentity.OutboxEvent) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CorrelationID == uuid.Nil {
		e.CorrelationID = uuid.New()
	}
	if e.EventVersion == 0 {
		e.EventVersion = 1
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO notification_outbox(id,enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key,correlation_id,originator_user_id,occurred_at) VALUES($1,$2,$3,$4,$5,NULLIF($6,''),NULLIF($7,''),$8,$9,$10,$11,$12) ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING`, e.ID, e.EnterpriseID, e.EventKey, e.EventVersion, e.AggregateType, e.AggregateInternalID, e.AggregatePublicID, e.Payload, e.DeduplicationKey, e.CorrelationID, e.OriginatorUserID, e.OccurredAt)
	return err
}

func (r *Repository) ListEvents(ctx context.Context) ([]notificationentity.Event, error) {
	rows, err := r.pool.Query(ctx, `SELECT event_key,version,name_pt_br,description_pt_br,module,event_kind,severity,allowed_cadences,template_key,enabled_by_default,suggested_recipient_roles,producer_status,producer_description FROM notification_event_catalog WHERE active ORDER BY module,event_key,version DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []notificationentity.Event{}
	for rows.Next() {
		var e notificationentity.Event
		var sev string
		var cads []string
		if err = rows.Scan(&e.Key, &e.Version, &e.Name, &e.Description, &e.Module, &e.EventKind, &sev, &cads, &e.TemplateKey, &e.EnabledByDefault, &e.SuggestedRecipientRoles, &e.ProducerStatus, &e.ProducerDescription); err != nil {
			return nil, err
		}
		e.Severity = notificationentity.Severity(sev)
		for _, c := range cads {
			e.AllowedCadences = append(e.AllowedCadences, notificationentity.Cadence(c))
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) ListEligibleUsers(ctx context.Context, tenant int64) ([]notificationentity.EligibleUser, error) {
	rows, err := r.pool.Query(ctx, `SELECT u.id,u.name,ue.role,u.is_active FROM user_enterprises ue JOIN users u ON u.id=ue.user_id WHERE ue.enterprise_id=$1 ORDER BY u.is_active DESC,u.name,u.id`, tenant)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []notificationentity.EligibleUser{}
	for rows.Next() {
		var v notificationentity.EligibleUser
		if err = rows.Scan(&v.ID, &v.Name, &v.Role, &v.Active); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) ListEligibleDepartments(ctx context.Context, tenant int64) ([]notificationentity.EligibleDepartment, error) {
	rows, err := r.pool.Query(ctx, `SELECT code,name,active FROM enterprise_departments WHERE enterprise_id=$1 ORDER BY active DESC,name,code`, tenant)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []notificationentity.EligibleDepartment{}
	for rows.Next() {
		var v notificationentity.EligibleDepartment
		if err = rows.Scan(&v.Code, &v.Description, &v.Active); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) GetSettings(ctx context.Context, tenant int64) (notificationentity.Settings, error) {
	var s notificationentity.Settings
	var digest time.Time
	err := r.pool.QueryRow(ctx, `SELECT enterprise_id,enabled,CURRENT_DATE+digest_time,timezone,retention_days,max_attachment_bytes,max_emails_per_minute,fiscal_config_id FROM enterprise_notification_settings WHERE enterprise_id=$1`, tenant).Scan(&s.EnterpriseID, &s.Enabled, &digest, &s.Timezone, &s.RetentionDays, &s.MaxAttachmentBytes, &s.MaxEmailsPerMinute, &s.FiscalConfigID)
	if errors.Is(err, pgx.ErrNoRows) {
		return notificationentity.Settings{EnterpriseID: tenant, DigestTime: "08:00", Timezone: "America/Sao_Paulo", RetentionDays: 365, MaxAttachmentBytes: 10 * 1024 * 1024, MaxEmailsPerMinute: 60}, nil
	}
	s.DigestTime = digest.Format("15:04")
	return s, err
}

func (r *Repository) SaveSettings(ctx context.Context, s notificationentity.Settings, actor uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO enterprise_notification_settings(enterprise_id,enabled,digest_time,timezone,retention_days,max_attachment_bytes,max_emails_per_minute,fiscal_config_id,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(enterprise_id) DO UPDATE SET enabled=EXCLUDED.enabled,digest_time=EXCLUDED.digest_time,timezone=EXCLUDED.timezone,retention_days=EXCLUDED.retention_days,max_attachment_bytes=EXCLUDED.max_attachment_bytes,max_emails_per_minute=EXCLUDED.max_emails_per_minute,fiscal_config_id=EXCLUDED.fiscal_config_id,updated_by=EXCLUDED.updated_by,updated_at=NOW()`, s.EnterpriseID, s.Enabled, s.DigestTime, s.Timezone, s.RetentionDays, s.MaxAttachmentBytes, s.MaxEmailsPerMinute, s.FiscalConfigID, actor)
	return err
}

func (r *Repository) ListSubscriptions(ctx context.Context, tenant int64) ([]notificationentity.Subscription, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,event_key,event_version,enabled,cadence,thresholds FROM enterprise_notification_subscriptions WHERE enterprise_id=$1 ORDER BY event_key`, tenant)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []notificationentity.Subscription{}
	for rows.Next() {
		var s notificationentity.Subscription
		var cadence string
		if err = rows.Scan(&s.ID, &s.EventKey, &s.EventVersion, &s.Enabled, &cadence, &s.Thresholds); err != nil {
			return nil, err
		}
		s.EnterpriseID = tenant
		s.Cadence = notificationentity.Cadence(cadence)
		rr, e := r.pool.Query(ctx, `SELECT recipient_type,user_id,COALESCE(recipient_key,'') FROM enterprise_notification_recipients WHERE enterprise_id=$1 AND subscription_id=$2`, tenant, s.ID)
		if e != nil {
			return nil, e
		}
		for rr.Next() {
			var rec notificationentity.Recipient
			var typ string
			if e = rr.Scan(&typ, &rec.UserID, &rec.Key); e != nil {
				rr.Close()
				return nil, e
			}
			rec.Type = notificationentity.RecipientType(typ)
			s.Recipients = append(s.Recipients, rec)
		}
		e = rr.Err()
		rr.Close()
		if e != nil {
			return nil, e
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) SaveSubscription(ctx context.Context, s notificationentity.Subscription, actor uuid.UUID) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	var cadenceAllowed bool
	err = tx.QueryRow(ctx, `SELECT $3=ANY(allowed_cadences) FROM notification_event_catalog WHERE event_key=$1 AND version=$2 AND active`, s.EventKey, s.EventVersion, s.Cadence).Scan(&cadenceAllowed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("%w: evento inexistente", notificationentity.ErrValidation)
		}
		return uuid.Nil, err
	}
	if !cadenceAllowed {
		return uuid.Nil, fmt.Errorf("%w: cadência não permitida para o evento", notificationentity.ErrValidation)
	}
	if s.ID == uuid.Nil {
		err = tx.QueryRow(ctx, `INSERT INTO enterprise_notification_subscriptions(enterprise_id,event_key,event_version,enabled,cadence,thresholds,created_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, s.EnterpriseID, s.EventKey, s.EventVersion, s.Enabled, s.Cadence, s.Thresholds, actor).Scan(&s.ID)
	} else {
		tag, e := tx.Exec(ctx, `UPDATE enterprise_notification_subscriptions SET event_key=$3,event_version=$4,enabled=$5,cadence=$6,thresholds=$7,updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, s.ID, s.EnterpriseID, s.EventKey, s.EventVersion, s.Enabled, s.Cadence, s.Thresholds)
		err = e
		if e == nil && tag.RowsAffected() != 1 {
			err = pgx.ErrNoRows
		}
	}
	if err != nil {
		return uuid.Nil, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM enterprise_notification_recipients WHERE enterprise_id=$1 AND subscription_id=$2`, s.EnterpriseID, s.ID); err != nil {
		return uuid.Nil, err
	}
	for _, rec := range s.Recipients {
		if rec.Type == notificationentity.RecipientUser {
			var ok bool
			err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_enterprises ue JOIN users u ON u.id=ue.user_id WHERE ue.enterprise_id=$1 AND ue.user_id=$2 AND u.is_active)`, s.EnterpriseID, rec.UserID).Scan(&ok)
			if err != nil || !ok {
				if err == nil {
					err = fmt.Errorf("%w: usuário inativo ou fora da empresa", notificationentity.ErrValidation)
				}
				return uuid.Nil, err
			}
		} else if rec.Type == notificationentity.RecipientDepartment {
			var ok bool
			err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM enterprise_departments WHERE enterprise_id=$1 AND code=$2 AND active)`, s.EnterpriseID, rec.Key).Scan(&ok)
			if err != nil || !ok {
				if err == nil {
					err = fmt.Errorf("%w: departamento inexistente ou inativo", notificationentity.ErrValidation)
				}
				return uuid.Nil, err
			}
		}
		_, err = tx.Exec(ctx, `INSERT INTO enterprise_notification_recipients(enterprise_id,subscription_id,recipient_type,user_id,recipient_key) VALUES($1,$2,$3,$4,NULLIF($5,''))`, s.EnterpriseID, s.ID, rec.Type, rec.UserID, rec.Key)
		if err != nil {
			return uuid.Nil, err
		}
	}
	return s.ID, tx.Commit(ctx)
}

func (r *Repository) DeleteSubscription(ctx context.Context, tenant int64, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM enterprise_notification_subscriptions WHERE enterprise_id=$1 AND id=$2`, tenant, id)
	if err == nil && tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return err
}

func (r *Repository) ListRecords(ctx context.Context, tenant int64, kind string, limit, offset int) ([]map[string]any, error) {
	var query string
	switch kind {
	case "alerts":
		query = `SELECT id,event_key,state,severity,summary,opened_at,last_seen_at,resolved_at FROM notification_alerts WHERE enterprise_id=$1 ORDER BY opened_at DESC LIMIT $2 OFFSET $3`
	case "deliveries":
		query = `SELECT id,state,channel,recipient_user_id,recipient_email_snapshot,subject_snapshot,attempts,sent_at,created_at,last_error_code FROM notification_deliveries WHERE enterprise_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	case "dead_letters":
		query = `SELECT id,delivery_id,reason_code,sanitized_reason,created_at,retried_at FROM notification_dead_letters WHERE enterprise_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	default:
		return nil, errors.New("consulta inválida")
	}
	rows, err := r.pool.Query(ctx, query, tenant, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	fields := rows.FieldDescriptions()
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		m := map[string]any{}
		for i, f := range fields {
			m[string(f.Name)] = vals[i]
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repository) RetryDelivery(ctx context.Context, tenant int64, id, actor uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var attempts int
	err = tx.QueryRow(ctx, `UPDATE notification_deliveries SET state='PENDENTE',next_attempt_at=NOW(),lease_owner=NULL,lease_until=NULL WHERE enterprise_id=$1 AND id=$2 AND state IN('FALHOU','DESCARTADO') RETURNING attempts`, tenant, id).Scan(&attempts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM notification_deliveries WHERE enterprise_id=$1 AND id=$2)`, tenant, id).Scan(&exists); e != nil {
				return e
			}
			if !exists {
				return fmt.Errorf("%w: entrega", notificationentity.ErrNotFound)
			}
			return fmt.Errorf("%w: entrega não está em estado reenviável", notificationentity.ErrConflict)
		}
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO notification_delivery_attempts(enterprise_id,delivery_id,attempt_number,outcome,manual_retry_by) VALUES($1,$2,$3,'PROCESSANDO',$4) ON CONFLICT(delivery_id,attempt_number) DO UPDATE SET manual_retry_by=EXCLUDED.manual_retry_by`, tenant, id, attempts+1, actor)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE notification_dead_letters SET retried_at=NOW(),retried_by=$3 WHERE enterprise_id=$1 AND delivery_id=$2`, tenant, id, actor)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) EnqueueTest(ctx context.Context, tenant int64, actor uuid.UUID) error {
	payload, _ := json.Marshal(map[string]any{"tipo": "teste", "gerado_em": time.Now().UTC()})
	_, err := r.pool.Exec(ctx, `INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,payload,deduplication_key,originator_user_id) VALUES($1,'OPERACAO_JOB_ESSENCIAL_SEM_EXECUCAO',1,'TESTE',$2,$3,$4)`, tenant, payload, fmt.Sprintf("teste:%s", uuid.NewString()), actor)
	return err
}

func (r *Repository) GetAlert(ctx context.Context, tenant int64, id uuid.UUID) (map[string]any, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,event_key,state,severity,summary,opened_at,last_seen_at,resolved_at FROM notification_alerts WHERE enterprise_id=$1 AND id=$2`, tenant, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, pgx.ErrNoRows
	}
	vals, err := rows.Values()
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for i, f := range rows.FieldDescriptions() {
		out[string(f.Name)] = vals[i]
	}
	return out, nil
}

func (r *Repository) ClaimOutbox(ctx context.Context, owner string, limit int, lease time.Duration) ([]notificationentity.OutboxEvent, error) {
	rows, err := r.pool.Query(ctx, `WITH candidates AS (
		SELECT id FROM notification_outbox WHERE
		(state IN ('PENDENTE','FALHOU') AND available_at<=NOW() AND next_attempt_at<=NOW()) OR (state='PROCESSANDO' AND lease_until<NOW())
		ORDER BY next_attempt_at,created_at FOR UPDATE SKIP LOCKED LIMIT $1
	) UPDATE notification_outbox o SET state='PROCESSANDO',lease_owner=$2,lease_until=NOW()+$3::interval
	FROM candidates c WHERE o.id=c.id
	RETURNING o.id,o.enterprise_id,o.event_key,o.event_version,o.aggregate_type,COALESCE(o.aggregate_internal_id,''),COALESCE(o.aggregate_public_id,''),o.payload,o.deduplication_key,o.correlation_id,o.originator_user_id,o.occurred_at,o.attempts`, limit, owner, lease.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []notificationentity.OutboxEvent{}
	for rows.Next() {
		var e notificationentity.OutboxEvent
		if err = rows.Scan(&e.ID, &e.EnterpriseID, &e.EventKey, &e.EventVersion, &e.AggregateType, &e.AggregateInternalID, &e.AggregatePublicID, &e.Payload, &e.DeduplicationKey, &e.CorrelationID, &e.OriginatorUserID, &e.OccurredAt, &e.Attempts); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) ProcessOutbox(ctx context.Context, e notificationentity.OutboxEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var severity, name, cadence, eventKind string
	var enabled bool
	err = tx.QueryRow(ctx, `SELECT c.severity,c.name_pt_br,s.cadence,st.enabled AND s.enabled,c.event_kind FROM notification_event_catalog c JOIN enterprise_notification_subscriptions s ON s.enterprise_id=$1 AND s.event_key=c.event_key AND s.event_version=c.version JOIN enterprise_notification_settings st ON st.enterprise_id=s.enterprise_id WHERE c.event_key=$2 AND c.version=$3`, e.EnterpriseID, e.EventKey, e.EventVersion).Scan(&severity, &name, &cadence, &enabled, &eventKind)
	if errors.Is(err, pgx.ErrNoRows) || !enabled {
		_, err = tx.Exec(ctx, `UPDATE notification_outbox SET state='CANCELADO',processed_at=NOW(),lease_owner=NULL,lease_until=NULL WHERE id=$1 AND enterprise_id=$2`, e.ID, e.EnterpriseID)
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if err != nil {
		return err
	}
	summary := name
	var p map[string]any
	if json.Unmarshal(e.Payload, &p) == nil {
		if v, ok := p["descricao"].(string); ok && strings.TrimSpace(v) != "" {
			summary = v
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO notification_alerts(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,cycle,deduplication_key,severity,summary,details) SELECT $1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),COALESCE(MAX(cycle),0)+1,$7,$8,$9,$10 FROM notification_alerts WHERE enterprise_id=$1 AND event_key=$2 AND aggregate_type=$4 AND COALESCE(aggregate_internal_id,'')=$5 ON CONFLICT(enterprise_id,event_key,deduplication_key) WHERE state='ABERTO' DO UPDATE SET last_seen_at=NOW(),summary=EXCLUDED.summary,details=EXCLUDED.details`, e.EnterpriseID, e.EventKey, e.EventVersion, e.AggregateType, e.AggregateInternalID, e.AggregatePublicID, e.DeduplicationKey, severity, summary, e.Payload)
	if err != nil {
		return err
	}
	if cadence == string(notificationentity.CadenceImmediate) || cadence == string(notificationentity.CadenceBoth) {
		_, err = tx.Exec(ctx, `WITH recipients AS (
		SELECT DISTINCT u.id,u.email,u.name FROM enterprise_notification_subscriptions s JOIN enterprise_notification_recipients r ON r.subscription_id=s.id AND r.enterprise_id=s.enterprise_id JOIN user_enterprises ue ON ue.enterprise_id=s.enterprise_id AND ((r.recipient_type='USUARIO' AND ue.user_id=r.user_id) OR (r.recipient_type='PAPEL' AND ue.role=r.recipient_key) OR (r.recipient_type='DEPARTAMENTO' AND EXISTS(SELECT 1 FROM enterprise_departments dep JOIN enterprise_department_users du ON du.enterprise_id=dep.enterprise_id AND du.department_id=dep.id WHERE dep.enterprise_id=s.enterprise_id AND dep.code=r.recipient_key AND dep.active AND du.user_id=ue.user_id))) JOIN users u ON u.id=ue.user_id AND u.is_active WHERE s.enterprise_id=$1 AND s.event_key=$2 AND s.enabled
		) INSERT INTO notification_deliveries(enterprise_id,outbox_id,recipient_user_id,recipient_email_snapshot,recipient_name_snapshot,subject_snapshot,message_id) SELECT $1,$3,id,email,name,$4,'<'||$3::text||'.'||id::text||'@venturerp.local>' FROM recipients ON CONFLICT(enterprise_id,message_id) DO NOTHING`, e.EnterpriseID, e.EventKey, e.ID, "[VentureERP] "+name)
		if err != nil {
			return err
		}
	}
	if eventKind == "EVENTO" && (cadence == string(notificationentity.CadenceImmediate) || cadence == string(notificationentity.CadenceBoth)) {
		_, err = tx.Exec(ctx, `UPDATE notification_alerts SET state='RESOLVIDO',resolved_at=NOW(),resolution_reason='Evento encaminhado para entrega imediata' WHERE enterprise_id=$1 AND event_key=$2 AND deduplication_key=$3 AND state='ABERTO'`, e.EnterpriseID, e.EventKey, e.DeduplicationKey)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE notification_outbox SET state='ENVIADO',processed_at=NOW(),lease_owner=NULL,lease_until=NULL WHERE id=$1 AND enterprise_id=$2`, e.ID, e.EnterpriseID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) FailOutbox(ctx context.Context, id string, attempt int, next time.Time, code, message string, discard bool) error {
	state := "FALHOU"
	if discard {
		state = "DESCARTADO"
	}
	_, err := r.pool.Exec(ctx, `UPDATE notification_outbox SET state=$2,attempts=$3,next_attempt_at=$4,last_error_code=$5,last_error_message=$6,lease_owner=NULL,lease_until=NULL WHERE id=$1`, id, state, attempt, next, code, sanitizeError(message))
	return err
}
func sanitizeError(v string) string {
	v = strings.ReplaceAll(strings.ReplaceAll(v, "\r", " "), "\n", " ")
	if len(v) > 500 {
		v = v[:500]
	}
	return v
}

type Delivery struct {
	ID             uuid.UUID
	EnterpriseID   int64
	OutboxID       *uuid.UUID
	Recipient      string
	RecipientName  string
	Subject        string
	MessageID      string
	Payload        json.RawMessage
	Attempts       int
	EnterpriseName string
	BrandColor     string
	LogoDataURI    string
	Attachments    []ports.EmailAttachment
}

func (r *Repository) ClaimDeliveries(ctx context.Context, owner string, limit int, lease time.Duration) ([]Delivery, error) {
	rows, err := r.pool.Query(ctx, `WITH c AS(SELECT id FROM notification_deliveries WHERE ((state IN('PENDENTE','FALHOU') AND next_attempt_at<=NOW()) OR (state='PROCESSANDO' AND lease_until<NOW())) ORDER BY next_attempt_at,created_at FOR UPDATE SKIP LOCKED LIMIT $1) UPDATE notification_deliveries d SET state='PROCESSANDO',lease_owner=$2,lease_until=NOW()+$3::interval FROM c WHERE d.id=c.id RETURNING d.id,d.enterprise_id,d.outbox_id,d.recipient_email_snapshot,d.recipient_name_snapshot,d.subject_snapshot,d.message_id,d.attempts`, limit, owner, lease.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Delivery{}
	for rows.Next() {
		var d Delivery
		if err = rows.Scan(&d.ID, &d.EnterpriseID, &d.OutboxID, &d.Recipient, &d.RecipientName, &d.Subject, &d.MessageID, &d.Attempts); err != nil {
			return nil, err
		}
		err = r.pool.QueryRow(ctx, `SELECT COALESCE(a.details||jsonb_build_object('_event_key',a.event_key,'_severity',c.severity,'_module',c.module,'_reminder_daily',TRUE),o.payload||jsonb_build_object('_event_key',o.event_key,'_severity',c.severity,'_module',c.module),jsonb_build_object('_event_key','RESUMO_DIARIO','_severity','INFORMATIVO','_module','OPERAÇÃO','descricao',COALESCE((SELECT string_agg(di.module_snapshot||' / '||di.severity_snapshot||' — '||di.summary_snapshot,E'\n' ORDER BY di.module_snapshot,di.severity_snapshot,di.created_at) FROM notification_digest_items di WHERE di.enterprise_id=d.enterprise_id AND di.digest_run_id=d.digest_run_id AND di.recipient_user_id=d.recipient_user_id),'Resumo diário'))),COALESCE(f.razao_social,e.name),COALESCE(f.brand_color,'#1F4E78'),CASE WHEN f.logo IS NOT NULL AND octet_length(f.logo)<=262144 AND f.logo_mime IN('image/png','image/jpeg') THEN 'data:'||f.logo_mime||';base64,'||encode(f.logo,'base64') ELSE '' END FROM notification_deliveries d LEFT JOIN notification_alerts a ON a.id=d.alert_id AND a.enterprise_id=d.enterprise_id LEFT JOIN notification_outbox o ON o.id=d.outbox_id LEFT JOIN notification_event_catalog c ON c.event_key=COALESCE(a.event_key,o.event_key) AND c.version=COALESCE(a.event_version,o.event_version) JOIN enterprise e ON e.id=d.enterprise_id LEFT JOIN enterprise_notification_settings st ON st.enterprise_id=d.enterprise_id LEFT JOIN fiscal_configs f ON f.id=st.fiscal_config_id WHERE d.id=$1 AND d.enterprise_id=$2`, d.ID, d.EnterpriseID).Scan(&d.Payload, &d.EnterpriseName, &d.BrandColor, &d.LogoDataURI)
		if err != nil {
			return nil, err
		}
		attachmentRows, attachmentErr := r.pool.Query(ctx, `SELECT file_name,mime_type,content,sha256 FROM notification_delivery_attachments WHERE enterprise_id=$1 AND delivery_id=$2 ORDER BY created_at,id`, d.EnterpriseID, d.ID)
		if attachmentErr != nil {
			return nil, attachmentErr
		}
		for attachmentRows.Next() {
			var attachment ports.EmailAttachment
			if attachmentErr = attachmentRows.Scan(&attachment.FileName, &attachment.MIMEType, &attachment.Content, &attachment.SHA256); attachmentErr != nil {
				attachmentRows.Close()
				return nil, attachmentErr
			}
			d.Attachments = append(d.Attachments, attachment)
		}
		attachmentErr = attachmentRows.Err()
		attachmentRows.Close()
		if attachmentErr != nil {
			return nil, attachmentErr
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// SchedulePendingReminders creates one rich, individual delivery per recipient and
// tenant-local day while a pending alert remains open. The message-id is the final
// idempotency barrier when multiple worker replicas evaluate the same alert.
func (r *Repository) SchedulePendingReminders(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `WITH due AS (
		SELECT a.id alert_id,a.enterprise_id,a.event_key,c.name_pt_br,s.id subscription_id,
		       st.timezone,(NOW() AT TIME ZONE st.timezone)::date local_date
		FROM notification_alerts a
		JOIN notification_event_catalog c ON c.event_key=a.event_key AND c.version=a.event_version AND c.event_kind='PENDENCIA'
		JOIN enterprise_notification_subscriptions s ON s.enterprise_id=a.enterprise_id AND s.event_key=a.event_key AND s.event_version=a.event_version AND s.enabled
		JOIN enterprise_notification_settings st ON st.enterprise_id=a.enterprise_id AND st.enabled
		WHERE a.state='ABERTO'
		  AND (NOW() AT TIME ZONE st.timezone)::time>=st.digest_time
		  AND (s.cadence='RESUMO_DIARIO' OR (a.opened_at AT TIME ZONE st.timezone)::date<(NOW() AT TIME ZONE st.timezone)::date)
	), recipients AS (
		SELECT DISTINCT due.*,u.id user_id,u.email,u.name
		FROM due
		JOIN enterprise_notification_recipients r ON r.subscription_id=due.subscription_id AND r.enterprise_id=due.enterprise_id
		JOIN user_enterprises ue ON ue.enterprise_id=due.enterprise_id AND ((r.recipient_type='USUARIO' AND ue.user_id=r.user_id) OR (r.recipient_type='PAPEL' AND ue.role=r.recipient_key) OR (r.recipient_type='DEPARTAMENTO' AND EXISTS(SELECT 1 FROM enterprise_departments dep JOIN enterprise_department_users du ON du.enterprise_id=dep.enterprise_id AND du.department_id=dep.id WHERE dep.enterprise_id=due.enterprise_id AND dep.code=r.recipient_key AND dep.active AND du.user_id=ue.user_id)))
		JOIN users u ON u.id=ue.user_id AND u.is_active
	)
	INSERT INTO notification_deliveries(enterprise_id,alert_id,recipient_user_id,recipient_email_snapshot,recipient_name_snapshot,subject_snapshot,message_id)
	SELECT enterprise_id,alert_id,user_id,email,name,'[VentureERP] Pendência: '||name_pt_br,'<reminder.'||alert_id::text||'.'||user_id::text||'.'||to_char(local_date,'YYYYMMDD')||'@venturerp.local>'
	FROM recipients ON CONFLICT(enterprise_id,message_id) DO NOTHING`)
	return err
}

func (r *Repository) ScheduleDigests(ctx context.Context) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO notification_digest_runs(enterprise_id,local_date,timezone,state,started_at)
	SELECT s.enterprise_id,(NOW() AT TIME ZONE s.timezone)::date,s.timezone,'PROCESSANDO',NOW() FROM enterprise_notification_settings s
	WHERE s.enabled AND (NOW() AT TIME ZONE s.timezone)::time>=s.digest_time
	AND EXISTS(SELECT 1 FROM notification_alerts a JOIN notification_event_catalog c ON c.event_key=a.event_key AND c.version=a.event_version AND c.event_kind='EVENTO' JOIN enterprise_notification_subscriptions sub ON sub.enterprise_id=a.enterprise_id AND sub.event_key=a.event_key AND sub.enabled AND sub.cadence IN('RESUMO_DIARIO','IMEDIATO_E_RESUMO_DIARIO') WHERE a.enterprise_id=s.enterprise_id AND a.state='ABERTO')
	ON CONFLICT(enterprise_id,local_date) DO NOTHING`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `WITH recipient_runs AS (
	SELECT DISTINCT run.id run_id,run.enterprise_id,u.id user_id,u.email,u.name FROM notification_digest_runs run
	JOIN notification_alerts a ON a.enterprise_id=run.enterprise_id AND a.state='ABERTO'
	JOIN notification_event_catalog c ON c.event_key=a.event_key AND c.version=a.event_version AND c.event_kind='EVENTO'
	JOIN enterprise_notification_subscriptions s ON s.enterprise_id=a.enterprise_id AND s.event_key=a.event_key AND s.enabled AND s.cadence IN('RESUMO_DIARIO','IMEDIATO_E_RESUMO_DIARIO')
	JOIN enterprise_notification_recipients r ON r.subscription_id=s.id AND r.enterprise_id=s.enterprise_id
	JOIN user_enterprises ue ON ue.enterprise_id=s.enterprise_id AND ((r.recipient_type='USUARIO' AND ue.user_id=r.user_id) OR (r.recipient_type='PAPEL' AND ue.role=r.recipient_key) OR (r.recipient_type='DEPARTAMENTO' AND EXISTS(SELECT 1 FROM enterprise_departments dep JOIN enterprise_department_users du ON du.enterprise_id=dep.enterprise_id AND du.department_id=dep.id WHERE dep.enterprise_id=s.enterprise_id AND dep.code=r.recipient_key AND dep.active AND du.user_id=ue.user_id)))
	JOIN users u ON u.id=ue.user_id AND u.is_active WHERE run.state='PROCESSANDO'
	) INSERT INTO notification_deliveries(enterprise_id,digest_run_id,recipient_user_id,recipient_email_snapshot,recipient_name_snapshot,subject_snapshot,message_id)
	SELECT enterprise_id,run_id,user_id,email,name,'[VentureERP] Resumo diário','<digest.'||run_id::text||'.'||user_id::text||'@venturerp.local>' FROM recipient_runs ON CONFLICT(enterprise_id,message_id) DO NOTHING`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `WITH recipient_alerts AS (
	SELECT DISTINCT run.id run_id,run.enterprise_id,u.id user_id,a.id alert_id,c.module,a.severity,a.summary FROM notification_digest_runs run JOIN notification_alerts a ON a.enterprise_id=run.enterprise_id AND a.state='ABERTO' JOIN notification_event_catalog c ON c.event_key=a.event_key AND c.version=a.event_version AND c.event_kind='EVENTO' JOIN enterprise_notification_subscriptions s ON s.enterprise_id=a.enterprise_id AND s.event_key=a.event_key AND s.enabled AND s.cadence IN('RESUMO_DIARIO','IMEDIATO_E_RESUMO_DIARIO') JOIN enterprise_notification_recipients r ON r.subscription_id=s.id AND r.enterprise_id=s.enterprise_id JOIN user_enterprises ue ON ue.enterprise_id=s.enterprise_id AND ((r.recipient_type='USUARIO' AND ue.user_id=r.user_id) OR (r.recipient_type='PAPEL' AND ue.role=r.recipient_key) OR (r.recipient_type='DEPARTAMENTO' AND EXISTS(SELECT 1 FROM enterprise_departments dep JOIN enterprise_department_users du ON du.enterprise_id=dep.enterprise_id AND du.department_id=dep.id WHERE dep.enterprise_id=s.enterprise_id AND dep.code=r.recipient_key AND dep.active AND du.user_id=ue.user_id))) JOIN users u ON u.id=ue.user_id AND u.is_active WHERE run.state='PROCESSANDO') INSERT INTO notification_digest_items(enterprise_id,digest_run_id,recipient_user_id,alert_id,module_snapshot,severity_snapshot,summary_snapshot) SELECT enterprise_id,run_id,user_id,alert_id,module,severity,summary FROM recipient_alerts ON CONFLICT DO NOTHING`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE notification_digest_runs run SET state='CANCELADO',completed_at=NOW() WHERE run.state='PROCESSANDO' AND NOT EXISTS(SELECT 1 FROM notification_deliveries d WHERE d.digest_run_id=run.id)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE notification_alerts a SET state='RESOLVIDO',resolved_at=NOW(),resolution_reason='Evento incluído no resumo diário' WHERE a.state='ABERTO' AND EXISTS(SELECT 1 FROM notification_digest_items di JOIN notification_event_catalog c ON c.event_key=a.event_key AND c.version=a.event_version WHERE di.alert_id=a.id AND c.event_kind='EVENTO')`)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) ScheduleOperationalAlerts(ctx context.Context) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	statements := []string{
		`INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key)
		 SELECT c.enterprise_id,'ESTOQUE_CONTAGEM_PROXIMA_VENCIMENTO',1,'CONTAGEM_CICLICA',c.id::text,c.id::text,jsonb_strip_nulls(jsonb_build_object('contagem_id',c.id,'descricao','Uma contagem cíclica está próxima do prazo e precisa ser preparada.','programada_para',c.scheduled_for,'item',jsonb_build_object('codigo',i.business_code,'descricao',i.name,'mascara',NULLIF(c.mask,''),'unidade_estoque',i.warehouse_unit_of_measurement::text,'classe_abc',i.planning_abc_class,'critico',i.planning_critical),'localizacao',jsonb_build_object('almoxarifado',w.code||' — '||COALESCE(w.description,w.code),'endereco_id',c.warehouse_address_id,'lote',c.lot_code))),'contagem:'||c.id::text||':proxima' FROM stock_cycle_counts c JOIN enterprise_notification_subscriptions s ON s.enterprise_id=c.enterprise_id AND s.event_key='ESTOQUE_CONTAGEM_PROXIMA_VENCIMENTO' AND s.enabled JOIN items i ON i.enterprise_id=c.enterprise_id AND i.code=c.item_code JOIN warehouse w ON w.id=c.warehouse_id WHERE c.state='PROGRAMADA' AND c.scheduled_for>NOW() AND c.scheduled_for<=NOW()+make_interval(days=>COALESCE((s.thresholds->>'antecedencia_dias')::int,3)) ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING`,
		`INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key)
		 SELECT c.enterprise_id,'ESTOQUE_CONTAGEM_VENCIDA',1,'CONTAGEM_CICLICA',c.id::text,c.id::text,jsonb_strip_nulls(jsonb_build_object('contagem_id',c.id,'descricao','A contagem cíclica venceu sem conclusão e o saldo ainda não foi confirmado.','programada_para',c.scheduled_for,'atraso_dias',GREATEST(EXTRACT(DAY FROM NOW()-c.scheduled_for)::int,0),'item',jsonb_build_object('codigo',i.business_code,'descricao',i.name,'mascara',NULLIF(c.mask,''),'unidade_estoque',i.warehouse_unit_of_measurement::text,'classe_abc',i.planning_abc_class,'critico',i.planning_critical),'localizacao',jsonb_build_object('almoxarifado',w.code||' — '||COALESCE(w.description,w.code),'endereco_id',c.warehouse_address_id,'lote',c.lot_code))),'contagem:'||c.id::text||':vencida' FROM stock_cycle_counts c JOIN items i ON i.enterprise_id=c.enterprise_id AND i.code=c.item_code JOIN warehouse w ON w.id=c.warehouse_id WHERE c.state='PROGRAMADA' AND c.scheduled_for<NOW() ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING`,
		`UPDATE notification_alerts a SET state='RESOLVIDO',resolved_at=NOW(),resolution_reason='Contagem iniciada ou encerrada' WHERE a.state='ABERTO' AND a.event_key IN('ESTOQUE_CONTAGEM_PROXIMA_VENCIMENTO','ESTOQUE_CONTAGEM_VENCIDA') AND NOT EXISTS(SELECT 1 FROM stock_cycle_counts c WHERE c.enterprise_id=a.enterprise_id AND c.id::text=a.aggregate_internal_id AND c.state='PROGRAMADA')`,
		`INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key)
		 SELECT b.enterprise_id,CASE WHEN b.quantity<0 THEN 'ESTOQUE_NEGATIVO' ELSE 'ESTOQUE_ABAIXO_MINIMO' END,1,'SALDO_ESTOQUE',b.id::text,i.business_code,
		 jsonb_strip_nulls(jsonb_build_object(
		   'saldo_id',b.id,
		   'descricao',CASE WHEN b.quantity<0 THEN 'O saldo físico do item está negativo e precisa de conferência imediata.' ELSE 'A disponibilidade do item ficou abaixo do estoque mínimo e pode comprometer o abastecimento.' END,
		   'item',jsonb_build_object('codigo',i.business_code,'codigo_interno',i.code,'descricao',i.name,'mascara',NULLIF(b.mask,''),'unidade_estoque',i.warehouse_unit_of_measurement::text,'unidade_compra',COALESCE(pref.uom,i.supplies_purchase_uom,i.warehouse_unit_of_measurement::text),'classe_abc',i.planning_abc_class,'critico',i.planning_critical),
		   'estoque',jsonb_build_object('almoxarifado',w.code||' — '||COALESCE(w.description,w.code),'saldo_fisico',b.quantity,'reservado',b.reserved_qty,'disponivel',b.available_qty,'minimo',b.minimum_stock,'seguranca',b.safety_stock,'maximo',b.maximum_stock,'necessidade_reposicao',GREATEST(b.minimum_stock-b.available_qty,0),'consumo_medio_mensal',i.warehouse_avg_monthly_consumption_manual,'cobertura_dias',CASE WHEN COALESCE(i.warehouse_avg_monthly_consumption_manual,0)>0 THEN ROUND((b.available_qty/i.warehouse_avg_monthly_consumption_manual::numeric)*30,1) END,'custo_medio',b.avg_cost,'valor_saldo',b.total_cost,'ultima_movimentacao',b.last_movement_at),
		   'fornecedor_recomendado',CASE WHEN pref.supplier_code IS NULL THEN NULL ELSE jsonb_build_object('codigo',pref.supplier_code,'nome',pref.supplier_name,'codigo_item_fornecedor',pref.supplier_item_code,'unidade_compra',COALESCE(pref.uom,i.supplies_purchase_uom),'lead_time_dias',pref.lead_time_days,'quantidade_embalagem',pref.package_quantity,'homologado',pref.homologated,'bloqueado',pref.blocked,'validade_cadastro',pref.valid_until) END,
		   'compras_em_aberto',open_po.orders
		 )),
		 'saldo:'||b.id::text||':ciclo:'||(COALESCE((SELECT MAX(a.cycle) FROM notification_alerts a WHERE a.enterprise_id=b.enterprise_id AND a.event_key=CASE WHEN b.quantity<0 THEN 'ESTOQUE_NEGATIVO' ELSE 'ESTOQUE_ABAIXO_MINIMO' END AND a.aggregate_internal_id=b.id::text),0)+1)::text
		 FROM stock_balances b
		 JOIN items i ON i.enterprise_id=b.enterprise_id AND i.code=b.item_code
		 JOIN warehouse w ON w.id=b.warehouse_id
		 LEFT JOIN LATERAL (
		   SELECT ips.supplier_code,COALESCE(s.trade_name,s.name) supplier_name,ips.supplier_item_code,ips.uom,ips.lead_time_days,ips.package_quantity,ips.valid_until,s.homologated,s.blocked
		   FROM item_preferred_suppliers ips JOIN suppliers s ON s.code=ips.supplier_code
		   WHERE ips.enterprise_id=b.enterprise_id AND ips.item_code=b.item_code AND ips.is_active AND s.is_active AND (ips.mask=b.mask OR ips.mask='')
		   ORDER BY (ips.mask=b.mask) DESC,ips.is_preferred DESC,ips.ranking,ips.id LIMIT 1
		 ) pref ON TRUE
		 LEFT JOIN LATERAL (
		   SELECT jsonb_agg(jsonb_build_object('pedido',x.order_number,'fornecedor',x.supplier_name,'quantidade_pendente',x.pending_qty,'unidade',COALESCE(x.purchase_uom,i.supplies_purchase_uom,i.warehouse_unit_of_measurement::text),'previsao',x.promised_date) ORDER BY x.promised_date NULLS LAST) orders
		   FROM (SELECT po.order_number,COALESCE(s.trade_name,s.name) supplier_name,(poi.requested_qty-poi.received_qty-poi.cancelled_qty) pending_qty,poi.purchase_uom,COALESCE(poi.promised_date,poi.delivery_date,po.delivery_date) promised_date
		         FROM enterprise e JOIN purchase_orders po ON po.enterprise_code=e.code JOIN purchase_order_items poi ON poi.purchase_order_code=po.code LEFT JOIN suppliers s ON s.code=po.supplier_code
		         WHERE e.id=b.enterprise_id AND poi.item_code=b.item_code AND poi.is_active AND po.is_active AND po.status NOT IN('CANCELLED','CANCELED','CLOSED') AND poi.requested_qty-poi.received_qty-poi.cancelled_qty>0
		         ORDER BY COALESCE(poi.promised_date,poi.delivery_date,po.delivery_date) NULLS LAST LIMIT 5) x
		 ) open_po ON TRUE
		 WHERE b.quantity<0 OR (b.minimum_stock>0 AND b.available_qty<b.minimum_stock)
		 ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING`,
		`UPDATE notification_alerts a SET state='RESOLVIDO',resolved_at=NOW(),resolution_reason='Saldo regularizado' WHERE a.state='ABERTO' AND a.event_key IN('ESTOQUE_NEGATIVO','ESTOQUE_ABAIXO_MINIMO') AND EXISTS(SELECT 1 FROM stock_balances b WHERE b.enterprise_id=a.enterprise_id AND b.id::text=a.aggregate_internal_id AND b.quantity>=0 AND (a.event_key<>'ESTOQUE_ABAIXO_MINIMO' OR b.minimum_stock<=0 OR b.available_qty>=b.minimum_stock))`,
		`INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key)
		 SELECT m.enterprise_id,'ESTOQUE_LOTE_PROXIMO_VENCIMENTO',1,'LOTE_ESTOQUE',m.id::text,COALESCE(m.lot,''),jsonb_build_object('movimentacao_id',m.id,'descricao','Há saldo disponível em lote próximo do vencimento. Avalie consumo, remanejamento ou bloqueio.','item',jsonb_build_object('codigo',i.business_code,'descricao',i.name,'unidade_estoque',i.warehouse_unit_of_measurement::text),'lote',jsonb_build_object('codigo',m.lot,'validade',m.expiration_date,'dias_ate_vencimento',m.expiration_date-CURRENT_DATE,'quantidade_disponivel',lb.quantity),'localizacao',jsonb_build_object('almoxarifado',w.code||' — '||COALESCE(w.description,w.code))),'lote_movimento:'||m.id::text FROM stock_movements m JOIN enterprise_notification_subscriptions s ON s.enterprise_id=m.enterprise_id AND s.event_key='ESTOQUE_LOTE_PROXIMO_VENCIMENTO' AND s.enabled JOIN items i ON i.enterprise_id=m.enterprise_id AND i.code=m.item_code JOIN warehouse w ON w.id=m.warehouse_id JOIN stock_lot_balances lb ON lb.enterprise_id=m.enterprise_id AND lb.item_code=m.item_code AND lb.warehouse_id=m.warehouse_id AND lb.lot=m.lot AND lb.quantity>0 WHERE m.lot IS NOT NULL AND m.expiration_date IS NOT NULL AND m.expiration_date BETWEEN CURRENT_DATE AND CURRENT_DATE+COALESCE((s.thresholds->>'antecedencia_dias')::int,30) ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING`,
		`INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key,originator_user_id,occurred_at)
	 SELECT m.enterprise_id,'ESTOQUE_MOVIMENTACAO_INCOMUM',1,'MOVIMENTACAO_ESTOQUE',m.id::text,m.id::text,jsonb_build_object('movimentacao_id',m.id,'item_codigo',m.item_code,'almoxarifado_id',m.warehouse_id,'tipo',m.movement_type,'quantidade',m.quantity,'valor_total',m.total_price,'regra',CASE WHEN ABS(m.quantity)>=COALESCE(NULLIF((s.thresholds->>'quantidade_limite')::numeric,0),1e30) THEN 'QUANTIDADE' WHEN ABS(m.total_price)>=COALESCE(NULLIF((s.thresholds->>'valor_limite')::numeric,0),1e30) THEN 'VALOR' ELSE 'HORARIO' END,'link','/stock/movements/'||m.id::text),'movimentacao:'||m.id::text,m.created_by,m.created_at FROM stock_movements m JOIN enterprise_notification_subscriptions s ON s.enterprise_id=m.enterprise_id AND s.event_key='ESTOQUE_MOVIMENTACAO_INCOMUM' AND s.enabled WHERE m.created_at>NOW()-INTERVAL '2 days' AND (ABS(m.quantity)>=COALESCE(NULLIF((s.thresholds->>'quantidade_limite')::numeric,0),1e30) OR ABS(m.total_price)>=COALESCE(NULLIF((s.thresholds->>'valor_limite')::numeric,0),1e30) OR ((s.thresholds ? 'horario_inicio') AND (s.thresholds ? 'horario_fim') AND m.created_at::time NOT BETWEEN (s.thresholds->>'horario_inicio')::time AND (s.thresholds->>'horario_fim')::time)) ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING`}
	for _, statement := range statements {
		if _, err = tx.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// SchedulePolicyCycleCounts materializes the permanent policy stored on each
// item into operational occurrences. The transaction-scoped advisory lock and
// the partial unique index make this safe when several API replicas run it.
func (r *Repository) SchedulePolicyCycleCounts(ctx context.Context) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var locked bool
	if err = tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(8420314)`).Scan(&locked); err != nil || !locked {
		return err
	}
	_, err = tx.Exec(ctx, `WITH policy AS (
		SELECT enterprise_id,code,warehouse_code,COALESCE(NULLIF(warehouse_cyclical_count_config->>'days','')::int,NULLIF(warehouse_cyclical_count_config->>'days_interval','')::int,NULLIF(warehouse_cyclical_count_config->>'DaysInterval','')::int,0) days
		FROM items
	), cancelled AS (
		UPDATE stock_cycle_counts c SET state='CANCELADA',updated_at=NOW()
		WHERE c.origin='POLITICA_ITEM' AND c.state='PROGRAMADA' AND NOT EXISTS (
			SELECT 1 FROM policy p WHERE p.enterprise_id=c.enterprise_id AND p.code=c.item_code AND p.warehouse_code=c.warehouse_id AND p.days>0 AND p.days=c.policy_days
		) RETURNING c.enterprise_id,c.id
	) INSERT INTO stock_cycle_count_audit(enterprise_id,cycle_count_id,action,previous_state,new_state,details)
	SELECT enterprise_id,id,'POLITICA_ALTERADA','PROGRAMADA','CANCELADA',jsonb_build_object('motivo','Política de contagem do item alterada ou desativada') FROM cancelled`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `WITH policy AS (
		SELECT i.enterprise_id,i.code item_code,i.warehouse_code warehouse_id,
		COALESCE(NULLIF(i.warehouse_cyclical_count_config->>'days','')::int,NULLIF(i.warehouse_cyclical_count_config->>'days_interval','')::int,NULLIF(i.warehouse_cyclical_count_config->>'DaysInterval','')::int,0) days,
		i.cyclical_count_policy_activated_at activated_at
		FROM items i JOIN warehouse w ON w.id=i.warehouse_code
	), due AS (
		SELECT p.*,COALESCE(last.approved_at,p.activated_at)+make_interval(days=>p.days) scheduled_for
		FROM policy p LEFT JOIN LATERAL (
			SELECT MAX(c.approved_at) approved_at FROM stock_cycle_counts c
			WHERE c.enterprise_id=p.enterprise_id AND c.item_code=p.item_code AND c.warehouse_id=p.warehouse_id AND c.state='APROVADA'
		) last ON TRUE WHERE p.days>0 AND p.activated_at IS NOT NULL
	), inserted AS (
		INSERT INTO stock_cycle_counts(enterprise_id,warehouse_id,item_code,mask,lot_code,scheduled_for,state,origin,policy_days)
		SELECT d.enterprise_id,d.warehouse_id,d.item_code,'','',d.scheduled_for,'PROGRAMADA','POLITICA_ITEM',d.days FROM due d
		WHERE NOT EXISTS (SELECT 1 FROM stock_cycle_counts c WHERE c.enterprise_id=d.enterprise_id AND c.warehouse_id=d.warehouse_id AND c.item_code=d.item_code AND c.warehouse_address_id IS NULL AND c.mask='' AND c.lot_code='' AND c.state IN('PROGRAMADA','EM_CONTAGEM','DIVERGENTE','CONCLUIDA'))
		ON CONFLICT DO NOTHING RETURNING enterprise_id,id,scheduled_for,policy_days
	) INSERT INTO stock_cycle_count_audit(enterprise_id,cycle_count_id,action,new_state,details)
	SELECT enterprise_id,id,'PROGRAMADA_AUTOMATICAMENTE','PROGRAMADA',jsonb_build_object('programada_para',scheduled_for,'intervalo_dias',policy_days,'origem','POLITICA_ITEM') FROM inserted`)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) CompleteDelivery(ctx context.Context, d Delivery, sendErr error) error {
	attempt := d.Attempts + 1
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if sendErr == nil {
		_, err = tx.Exec(ctx, `UPDATE notification_deliveries SET state='ENVIADO',attempts=$3,sent_at=NOW(),lease_owner=NULL,lease_until=NULL,last_error_code=NULL,last_error_message=NULL WHERE id=$1 AND enterprise_id=$2`, d.ID, d.EnterpriseID, attempt)
		if err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO notification_delivery_attempts(enterprise_id,delivery_id,attempt_number,finished_at,outcome) VALUES($1,$2,$3,NOW(),'ENVIADO') ON CONFLICT(delivery_id,attempt_number) DO UPDATE SET finished_at=EXCLUDED.finished_at,outcome=EXCLUDED.outcome,provider_code=NULL,sanitized_error=NULL`, d.EnterpriseID, d.ID, attempt)
		}
	} else {
		discard := attempt >= 6 || ports.FailureClass(sendErr) == ports.EmailFailurePermanent
		state := "FALHOU"
		if discard {
			state = "DESCARTADO"
		}
		msg := sanitizeError(sendErr.Error())
		_, err = tx.Exec(ctx, `UPDATE notification_deliveries SET state=$3,attempts=$4,next_attempt_at=$5,lease_owner=NULL,lease_until=NULL,last_error_code='PROVEDOR',last_error_message=$6 WHERE id=$1 AND enterprise_id=$2`, d.ID, d.EnterpriseID, state, attempt, time.Now().Add(notificationentity.RetryDelay(attempt)), msg)
		if err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO notification_delivery_attempts(enterprise_id,delivery_id,attempt_number,finished_at,outcome,provider_code,sanitized_error) VALUES($1,$2,$3,NOW(),$4,'PROVEDOR',$5) ON CONFLICT(delivery_id,attempt_number) DO UPDATE SET finished_at=EXCLUDED.finished_at,outcome=EXCLUDED.outcome,provider_code=EXCLUDED.provider_code,sanitized_error=EXCLUDED.sanitized_error`, d.EnterpriseID, d.ID, attempt, state, msg)
		}
		if err == nil && discard {
			_, err = tx.Exec(ctx, `INSERT INTO notification_dead_letters(enterprise_id,delivery_id,reason_code,sanitized_reason) VALUES($1,$2,'TENTATIVAS_ESGOTADAS',$3) ON CONFLICT(delivery_id) DO UPDATE SET reason_code=EXCLUDED.reason_code,sanitized_reason=EXCLUDED.sanitized_reason,created_at=NOW(),retried_at=NULL,retried_by=NULL`, d.EnterpriseID, d.ID, msg)
		}
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE notification_digest_runs run SET state=CASE WHEN EXISTS(SELECT 1 FROM notification_deliveries child WHERE child.digest_run_id=run.id AND child.state='DESCARTADO') THEN 'FALHOU' ELSE 'ENVIADO' END,completed_at=NOW() WHERE run.id=(SELECT digest_run_id FROM notification_deliveries WHERE id=$1) AND NOT EXISTS(SELECT 1 FROM notification_deliveries child WHERE child.digest_run_id=run.id AND child.state IN('PENDENTE','PROCESSANDO','FALHOU'))`, d.ID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type ProviderPermit struct {
	Allowed bool
	RetryAt time.Time
	Reason  string
}

func (r *Repository) AcquireProviderPermit(ctx context.Context, tenant int64, owner string) (ProviderPermit, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ProviderPermit{}, err
	}
	defer tx.Rollback(ctx)
	var state string
	var openedUntil, probeLease *time.Time
	var probeOwner *string
	err = tx.QueryRow(ctx, `SELECT state,opened_until,probe_owner,probe_lease_until FROM notification_provider_circuit WHERE provider_key='EMAIL_CENTRAL' FOR UPDATE`).Scan(&state, &openedUntil, &probeOwner, &probeLease)
	if err != nil {
		return ProviderPermit{}, err
	}
	now := time.Now().UTC()
	if state == "ABERTO" && openedUntil != nil && openedUntil.After(now) {
		return ProviderPermit{RetryAt: *openedUntil, Reason: "CIRCUITO_ABERTO"}, tx.Commit(ctx)
	}
	if state == "ABERTO" || state == "SEMIABERTO" {
		if state == "SEMIABERTO" && probeLease != nil && probeLease.After(now) && probeOwner != nil && *probeOwner != owner {
			return ProviderPermit{RetryAt: *probeLease, Reason: "CIRCUITO_SEMIABERTO"}, tx.Commit(ctx)
		}
		_, err = tx.Exec(ctx, `UPDATE notification_provider_circuit SET state='SEMIABERTO',probe_owner=$1,probe_lease_until=NOW()+INTERVAL '30 seconds',updated_at=NOW() WHERE provider_key='EMAIL_CENTRAL'`, owner)
		if err != nil {
			return ProviderPermit{}, err
		}
	}
	var tenantLimit int
	err = tx.QueryRow(ctx, `SELECT max_emails_per_minute FROM enterprise_notification_settings WHERE enterprise_id=$1`, tenant).Scan(&tenantLimit)
	if errors.Is(err, pgx.ErrNoRows) {
		tenantLimit = 60
		err = nil
	}
	if err != nil {
		return ProviderPermit{}, err
	}
	window := now.Truncate(time.Minute)
	acquire := func(scope string, limit int) (bool, error) {
		var count int
		e := tx.QueryRow(ctx, `INSERT INTO notification_provider_rate_windows(scope_key,window_start,sent_count) VALUES($1,$2,1) ON CONFLICT(scope_key,window_start) DO UPDATE SET sent_count=notification_provider_rate_windows.sent_count+1 WHERE notification_provider_rate_windows.sent_count<$3 RETURNING sent_count`, scope, window, limit).Scan(&count)
		if errors.Is(e, pgx.ErrNoRows) {
			return false, nil
		}
		return e == nil, e
	}
	ok, err := acquire("GLOBAL", 300)
	if err != nil {
		return ProviderPermit{}, err
	}
	if !ok {
		return ProviderPermit{RetryAt: window.Add(time.Minute), Reason: "LIMITE_GLOBAL"}, tx.Commit(ctx)
	}
	ok, err = acquire(fmt.Sprintf("EMPRESA:%d", tenant), tenantLimit)
	if err != nil {
		return ProviderPermit{}, err
	}
	if !ok {
		_, _ = tx.Exec(ctx, `UPDATE notification_provider_rate_windows SET sent_count=GREATEST(sent_count-1,0) WHERE scope_key='GLOBAL' AND window_start=$1`, window)
		return ProviderPermit{RetryAt: window.Add(time.Minute), Reason: "LIMITE_EMPRESA"}, tx.Commit(ctx)
	}
	return ProviderPermit{Allowed: true}, tx.Commit(ctx)
}

func (r *Repository) RecordProviderResult(ctx context.Context, success bool) error {
	if success {
		_, err := r.pool.Exec(ctx, `UPDATE notification_provider_circuit SET consecutive_failures=0,state='FECHADO',opened_until=NULL,probe_owner=NULL,probe_lease_until=NULL,updated_at=NOW() WHERE provider_key='EMAIL_CENTRAL'`)
		return err
	}
	_, err := r.pool.Exec(ctx, `UPDATE notification_provider_circuit SET consecutive_failures=consecutive_failures+1,state=CASE WHEN consecutive_failures+1>=5 THEN 'ABERTO' ELSE state END,opened_until=CASE WHEN consecutive_failures+1>=5 THEN NOW()+INTERVAL '2 minutes' ELSE opened_until END,probe_owner=NULL,probe_lease_until=NULL,updated_at=NOW() WHERE provider_key='EMAIL_CENTRAL'`)
	return err
}
func (r *Repository) DeferDelivery(ctx context.Context, tenant int64, id uuid.UUID, next time.Time, reason string) error {
	_, err := r.pool.Exec(ctx, `UPDATE notification_deliveries SET state='PENDENTE',next_attempt_at=$3,lease_owner=NULL,lease_until=NULL,last_error_code=$4,last_error_message=NULL WHERE enterprise_id=$1 AND id=$2`, tenant, id, next, reason)
	return err
}

func (r *Repository) Cleanup(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM notification_provider_rate_windows WHERE window_start<NOW()-INTERVAL '10 minutes';
	WITH policies AS(SELECT enterprise_id,retention_days FROM enterprise_notification_settings) DELETE FROM notification_delivery_attempts a USING policies p WHERE a.enterprise_id=p.enterprise_id AND a.finished_at<NOW()-(p.retention_days||' days')::interval AND NOT EXISTS(SELECT 1 FROM notification_dead_letters d WHERE d.delivery_id=a.delivery_id);
	WITH policies AS(SELECT enterprise_id,retention_days FROM enterprise_notification_settings) DELETE FROM notification_deliveries d USING policies p WHERE d.enterprise_id=p.enterprise_id AND d.state IN('ENVIADO','CANCELADO') AND d.created_at<NOW()-(p.retention_days||' days')::interval AND NOT EXISTS(SELECT 1 FROM notification_delivery_attempts a WHERE a.delivery_id=d.id) AND NOT EXISTS(SELECT 1 FROM notification_dead_letters dl WHERE dl.delivery_id=d.id);
	WITH policies AS(SELECT enterprise_id,retention_days FROM enterprise_notification_settings) DELETE FROM notification_digest_runs d USING policies p WHERE d.enterprise_id=p.enterprise_id AND d.state IN('ENVIADO','CANCELADO') AND d.created_at<NOW()-(p.retention_days||' days')::interval AND NOT EXISTS(SELECT 1 FROM notification_deliveries x WHERE x.digest_run_id=d.id);
	WITH policies AS(SELECT enterprise_id,retention_days FROM enterprise_notification_settings) DELETE FROM notification_alerts a USING policies p WHERE a.enterprise_id=p.enterprise_id AND a.state IN('RESOLVIDO','IGNORADO') AND COALESCE(a.resolved_at,a.last_seen_at)<NOW()-(p.retention_days||' days')::interval AND NOT EXISTS(SELECT 1 FROM notification_digest_items i WHERE i.alert_id=a.id);
	WITH policies AS(SELECT enterprise_id,retention_days FROM enterprise_notification_settings) DELETE FROM notification_outbox o USING policies p WHERE o.enterprise_id=p.enterprise_id AND o.state IN('ENVIADO','CANCELADO') AND o.created_at<NOW()-(p.retention_days||' days')::interval AND NOT EXISTS(SELECT 1 FROM notification_deliveries d WHERE d.outbox_id=o.id);`)
	return err
}

const cycleColumns = `id,enterprise_id,warehouse_id,warehouse_address_id,(SELECT i.business_code FROM items i WHERE i.enterprise_id=stock_cycle_counts.enterprise_id AND i.code=stock_cycle_counts.item_code),item_code,mask,lot_code,scheduled_for,state,origin,policy_days,expected_quantity::text,counted_quantity::text,divergence_quantity::text,counted_by,approved_by,started_at,completed_at,approved_at,created_at,updated_at`

type rowScanner interface{ Scan(...any) error }

func nullableNumeric(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func scanCycle(row rowScanner) (notificationentity.CycleCount, error) {
	var c notificationentity.CycleCount
	var state, origin string
	var expected, counted, divergence *string
	err := row.Scan(&c.ID, &c.EnterpriseID, &c.WarehouseID, &c.WarehouseAddressID, &c.ItemCode, &c.LegacyItemCode, &c.Mask, &c.LotCode, &c.ScheduledFor, &state, &origin, &c.PolicyDays, &expected, &counted, &divergence, &c.CountedBy, &c.ApprovedBy, &c.StartedAt, &c.CompletedAt, &c.ApprovedAt, &c.CreatedAt, &c.UpdatedAt)
	c.State = notificationentity.CycleCountState(state)
	c.Origin = notificationentity.CycleCountOrigin(origin)
	parse := func(v *string) *decimal.Decimal {
		if v == nil {
			return nil
		}
		d, e := decimal.NewFromString(*v)
		if e != nil {
			return nil
		}
		return &d
	}
	c.ExpectedQuantity = parse(expected)
	c.CountedQuantity = parse(counted)
	c.DivergenceQuantity = parse(divergence)
	return c, err
}
func (r *Repository) CreateCycleCount(ctx context.Context, c notificationentity.CycleCount, actor uuid.UUID) (notificationentity.CycleCount, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return notificationentity.CycleCount{}, err
	}
	defer tx.Rollback(ctx)
	var itemID int64
	err = tx.QueryRow(ctx, `SELECT code FROM items WHERE enterprise_id=$1 AND business_code=upper(btrim($2))`, c.EnterpriseID, c.ItemCode).Scan(&itemID)
	if errors.Is(err, pgx.ErrNoRows) {
		return notificationentity.CycleCount{}, fmt.Errorf("%w: item %q não encontrado", notificationentity.ErrValidation, c.ItemCode)
	}
	if err != nil {
		return notificationentity.CycleCount{}, err
	}
	var associationsValid bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM warehouse WHERE id=$1)`, c.WarehouseID).Scan(&associationsValid)
	if err != nil {
		return notificationentity.CycleCount{}, err
	}
	if !associationsValid {
		return notificationentity.CycleCount{}, fmt.Errorf("%w: almoxarifado inválido", notificationentity.ErrValidation)
	}
	var createdID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO stock_cycle_counts(enterprise_id,warehouse_id,warehouse_address_id,item_code,mask,lot_code,scheduled_for,state) VALUES($1,$2,$3,$4,$5,$6,$7,'PROGRAMADA') RETURNING id`, c.EnterpriseID, c.WarehouseID, c.WarehouseAddressID, itemID, c.Mask, c.LotCode, c.ScheduledFor).Scan(&createdID)
	if err != nil {
		return notificationentity.CycleCount{}, err
	}
	created, err := scanCycle(tx.QueryRow(ctx, `SELECT `+cycleColumns+` FROM stock_cycle_counts WHERE enterprise_id=$1 AND id=$2`, c.EnterpriseID, createdID))
	if err != nil {
		return notificationentity.CycleCount{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO stock_cycle_count_audit(enterprise_id,cycle_count_id,action,new_state,actor_user_id,details) VALUES($1,$2,'PROGRAMACAO','PROGRAMADA',$3,jsonb_build_object('programada_para',$4::timestamptz))`, c.EnterpriseID, created.ID, actor, created.ScheduledFor)
	if err != nil {
		return notificationentity.CycleCount{}, err
	}
	return created, tx.Commit(ctx)
}
func (r *Repository) ListCycleCounts(ctx context.Context, tenant int64, limit, offset int) ([]notificationentity.CycleCount, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cycleColumns+` FROM stock_cycle_counts WHERE enterprise_id=$1 ORDER BY scheduled_for DESC,id LIMIT $2 OFFSET $3`, tenant, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []notificationentity.CycleCount{}
	for rows.Next() {
		c, e := scanCycle(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (r *Repository) GetCycleCount(ctx context.Context, tenant int64, id uuid.UUID) (notificationentity.CycleCount, error) {
	v, err := scanCycle(r.pool.QueryRow(ctx, `SELECT `+cycleColumns+` FROM stock_cycle_counts WHERE enterprise_id=$1 AND id=$2`, tenant, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return v, fmt.Errorf("%w: contagem cíclica", notificationentity.ErrNotFound)
	}
	return v, err
}
func (r *Repository) TransitionCycleCount(ctx context.Context, tenant int64, id, actor uuid.UUID, target notificationentity.CycleCountState, countedText *string, reason string) (notificationentity.CycleCount, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return notificationentity.CycleCount{}, err
	}
	defer tx.Rollback(ctx)
	current, err := scanCycle(tx.QueryRow(ctx, `SELECT `+cycleColumns+` FROM stock_cycle_counts WHERE enterprise_id=$1 AND id=$2 FOR UPDATE`, tenant, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notificationentity.CycleCount{}, fmt.Errorf("%w: contagem cíclica", notificationentity.ErrNotFound)
		}
		return notificationentity.CycleCount{}, err
	}
	if !notificationentity.CanTransition(current.State, target) {
		return notificationentity.CycleCount{}, fmt.Errorf("%w: transição de contagem inválida: %s para %s", notificationentity.ErrConflict, current.State, target)
	}
	actualTarget := target
	var counted *decimal.Decimal
	if countedText != nil {
		value, e := decimal.NewFromString(*countedText)
		if e != nil || value.IsNegative() {
			return notificationentity.CycleCount{}, fmt.Errorf("%w: quantidade contada inválida", notificationentity.ErrValidation)
		}
		counted = &value
	}
	var expected decimal.Decimal
	if current.ExpectedQuantity != nil {
		expected = *current.ExpectedQuantity
	} else {
		query := `SELECT COALESCE(quantity,0)::text FROM stock_balances WHERE enterprise_id=$1 AND warehouse_id=$2 AND item_code=$3 AND mask=$4`
		args := []any{tenant, current.WarehouseID, current.LegacyItemCode, current.Mask}
		if current.LotCode != "" {
			query = `SELECT COALESCE(quantity,0)::text FROM stock_lot_balances WHERE enterprise_id=$1 AND warehouse_id=$2 AND item_code=$3 AND mask=$4 AND lot=$5`
			args = append(args, current.LotCode)
		}
		var raw string
		if e := tx.QueryRow(ctx, query, args...).Scan(&raw); errors.Is(e, pgx.ErrNoRows) {
			raw = "0"
		} else if e != nil {
			return notificationentity.CycleCount{}, e
		}
		expected, _ = decimal.NewFromString(raw)
	}
	if counted != nil {
		div := counted.Sub(expected)
		var toleranceText string
		err = tx.QueryRow(ctx, `SELECT COALESCE((thresholds->'tolerancia_por_item'->>$2)::numeric,(thresholds->'tolerancia_por_almoxarifado'->>$3)::numeric,(thresholds->>'tolerancia_quantidade')::numeric,0)::text FROM enterprise_notification_subscriptions WHERE enterprise_id=$1 AND event_key='ESTOQUE_CONTAGEM_DIVERGENCIA' AND enabled`, tenant, current.ItemCode, strconv.FormatInt(current.WarehouseID, 10)).Scan(&toleranceText)
		if errors.Is(err, pgx.ErrNoRows) {
			toleranceText = "0"
		} else if err != nil {
			return notificationentity.CycleCount{}, err
		}
		tolerance, toleranceErr := decimal.NewFromString(toleranceText)
		if toleranceErr != nil || tolerance.IsNegative() {
			return notificationentity.CycleCount{}, errors.New("tolerância de contagem inválida")
		}
		current.CountedQuantity = counted
		current.DivergenceQuantity = &div
		if target == notificationentity.CycleCompleted && div.Abs().GreaterThan(tolerance) {
			actualTarget = notificationentity.CycleDivergent
		}
	}
	set := `state=$3,expected_quantity=COALESCE(expected_quantity,$4::numeric),updated_at=NOW()`
	args := []any{tenant, id, string(actualTarget), expected.String()}
	switch actualTarget {
	case notificationentity.CycleCounting:
		set += `,started_at=COALESCE(started_at,NOW()),counted_by=$5`
		args = append(args, actor)
	case notificationentity.CycleDivergent:
		set += `,counted_quantity=$5::numeric,divergence_quantity=$6::numeric,counted_by=$7`
		args = append(args, current.CountedQuantity.String(), current.DivergenceQuantity.String(), actor)
	case notificationentity.CycleCompleted:
		set += `,counted_quantity=COALESCE($5::numeric,counted_quantity),divergence_quantity=COALESCE($6::numeric,divergence_quantity),completed_at=NOW(),counted_by=COALESCE(counted_by,$7)`
		countedValue, divergenceValue := "", ""
		if current.CountedQuantity != nil {
			countedValue = current.CountedQuantity.String()
		}
		if current.DivergenceQuantity != nil {
			divergenceValue = current.DivergenceQuantity.String()
		}
		args = append(args, nullableNumeric(countedValue), nullableNumeric(divergenceValue), actor)
	case notificationentity.CycleApproved:
		set += `,approved_at=NOW(),approved_by=$5`
		args = append(args, actor)
	}
	updated, err := scanCycle(tx.QueryRow(ctx, `UPDATE stock_cycle_counts SET `+set+` WHERE enterprise_id=$1 AND id=$2 RETURNING `+cycleColumns, args...))
	if err != nil {
		return notificationentity.CycleCount{}, fmt.Errorf("atualizar contagem (%T,%T,%T,%T): %w", args[0], args[1], args[2], args[3], err)
	}
	if actualTarget == notificationentity.CycleCompleted || actualTarget == notificationentity.CycleApproved {
		_, err = tx.Exec(ctx, `UPDATE notification_alerts SET state='RESOLVIDO',resolved_at=NOW(),resolution_reason='Contagem concluída' WHERE enterprise_id=$1 AND event_key='ESTOQUE_CONTAGEM_DIVERGENCIA' AND aggregate_internal_id=$2 AND state='ABERTO'`, tenant, id.String())
		if err != nil {
			return notificationentity.CycleCount{}, err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO stock_cycle_count_audit(enterprise_id,cycle_count_id,action,previous_state,new_state,actor_user_id,details) VALUES($1,$2,$3,$4,$5,$6,jsonb_build_object('motivo',NULLIF($7,'')))`, tenant, id, "TRANSICAO", string(current.State), string(actualTarget), actor, reason)
	if err != nil {
		return notificationentity.CycleCount{}, err
	}
	eventKey := ""
	switch actualTarget {
	case notificationentity.CycleDivergent:
		eventKey = "ESTOQUE_CONTAGEM_DIVERGENCIA"
	case notificationentity.CycleCompleted:
		eventKey = "ESTOQUE_CONTAGEM_CONCLUIDA"
	case notificationentity.CycleApproved:
		eventKey = "ESTOQUE_CONTAGEM_APROVADA"
	}
	if eventKey != "" {
		payload, _ := json.Marshal(map[string]any{"contagem_id": id, "almoxarifado_id": updated.WarehouseID, "endereco_id": updated.WarehouseAddressID, "item_codigo": updated.ItemCode, "mascara": updated.Mask, "lote": updated.LotCode, "quantidade_esperada": expected.String(), "quantidade_contada": func() string {
			if updated.CountedQuantity == nil {
				return ""
			}
			return updated.CountedQuantity.String()
		}(), "divergencia": func() string {
			if updated.DivergenceQuantity == nil {
				return ""
			}
			return updated.DivergenceQuantity.String()
		}(), "link": "/stock/cycle-counts/" + id.String()})
		_, err = tx.Exec(ctx, `INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key,originator_user_id) VALUES($1,$2,1,'CONTAGEM_CICLICA',$3,$3,$4,$5,$6) ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING`, tenant, eventKey, id.String(), payload, id.String()+":"+string(actualTarget), actor)
		if err != nil {
			return notificationentity.CycleCount{}, err
		}
	}
	return updated, tx.Commit(ctx)
}
