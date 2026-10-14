package planning_run

import (
	"context"
	"sync"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_uc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// lockNamespace separa os cadeados do planejamento de qualquer outro uso de
// advisory lock no banco.
const lockNamespace = 4210

type Repository struct {
	pool *pgxpool.Pool
	// locks guarda a conexão que detém o cadeado de cada empresa, porque o
	// advisory lock só pode ser liberado pela conexão que o tomou.
	mu    sync.Mutex
	locks map[int64]*pgxpool.Conn
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, locks: map[int64]*pgxpool.Conn{}}
}

// colunasSettings é o que DueSettings e Settings leem, na ordem de scanSettings.
const colunasSettings = `s.enterprise_id, e.code, s.is_enabled, s.run_hour, s.run_minute,
		       COALESCE(s.plan_code, 0), s.initial_order_number, s.generate_llc,
		       s.last_snapshot_at, s.last_run_at, COALESCE(s.last_run_status, ''), s.updated_by`

func scanSettings(row pgx.Row, s *planning_uc.AutoRunSettings) error {
	return row.Scan(&s.EnterpriseID, &s.EnterpriseCode, &s.IsEnabled, &s.RunHour, &s.RunMinute,
		&s.PlanCode, &s.InitialOrderNumber, &s.GenerateLLC,
		&s.LastSnapshotAt, &s.LastRunAt, &s.LastRunStatus, &s.UpdatedBy)
}

// DueSettings devolve as empresas cuja janela já passou hoje e que ainda não
// rodaram desde então. "Hoje" e "a hora da janela" são os do fuso da empresa
// (o mesmo do calendário das máquinas), não os do processo nem os da sessão
// do banco. Só entra quem tem responsável: sem ele não há em nome de quem
// gravar as ordens.
func (r *Repository) DueSettings(ctx context.Context, now time.Time) ([]planning_uc.AutoRunSettings, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+colunasSettings+`
		  FROM planning_auto_run_settings s
		  JOIN enterprise e ON e.id = s.enterprise_id
		 CROSS JOIN LATERAL (
		       SELECT COALESCE((SELECT NULLIF(n.timezone, '') FROM enterprise_notification_settings n
		                         WHERE n.enterprise_id = s.enterprise_id), 'America/Sao_Paulo') AS tz) fuso
		 CROSS JOIN LATERAL (SELECT $1::timestamptz AT TIME ZONE fuso.tz AS agora) local
		 WHERE s.is_enabled
		   AND s.plan_code IS NOT NULL
		   AND s.updated_by IS NOT NULL
		   AND local.agora::time >= make_time(s.run_hour, s.run_minute, 0)
		   AND (s.last_run_at IS NULL OR (s.last_run_at AT TIME ZONE fuso.tz)::date < local.agora::date)`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []planning_uc.AutoRunSettings{}
	for rows.Next() {
		var s planning_uc.AutoRunSettings
		if err := scanSettings(rows, &s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) Settings(ctx context.Context, enterpriseID int64) (*planning_uc.AutoRunSettings, error) {
	var s planning_uc.AutoRunSettings
	err := scanSettings(r.pool.QueryRow(ctx, `
		SELECT `+colunasSettings+`
		  FROM planning_auto_run_settings s
		  JOIN enterprise e ON e.id = s.enterprise_id
		 WHERE s.enterprise_id = $1`, enterpriseID), &s)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &planning_uc.AutoRunSettings{EnterpriseID: enterpriseID, RunHour: 2, InitialOrderNumber: 1, GenerateLLC: true}, nil
		}
		return nil, err
	}
	return &s, nil
}

// SaveSettings grava a parametrização sem mexer no corte nem na última
// execução — esses só a própria execução atualiza.
func (r *Repository) SaveSettings(ctx context.Context, s planning_uc.AutoRunSettings) error {
	var plano *int64
	if s.PlanCode > 0 {
		plano = &s.PlanCode
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO planning_auto_run_settings
		    (enterprise_id, is_enabled, run_hour, run_minute, plan_code,
		     initial_order_number, generate_llc, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (enterprise_id) DO UPDATE
		   SET is_enabled           = EXCLUDED.is_enabled,
		       run_hour             = EXCLUDED.run_hour,
		       run_minute           = EXCLUDED.run_minute,
		       plan_code            = EXCLUDED.plan_code,
		       initial_order_number = EXCLUDED.initial_order_number,
		       generate_llc         = EXCLUDED.generate_llc,
		       updated_by           = EXCLUDED.updated_by,
		       updated_at           = now()`,
		s.EnterpriseID, s.IsEnabled, s.RunHour, s.RunMinute, plano,
		s.InitialOrderNumber, s.GenerateLLC, s.UpdatedBy)
	return err
}

// PlanExists confere se o plano é da empresa — o plano de outra empresa não
// pode ser escolhido nem por engano — e se é um plano real (não de cenário).
func (r *Repository) PlanExists(ctx context.Context, enterpriseID, planCode int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM production_plans WHERE code = $1 AND enterprise_id = $2 AND is_active AND NOT is_scenario)`,
		planCode, enterpriseID).Scan(&ok)
	return ok, err
}

// ListRuns devolve as últimas execuções da empresa com o nome de quem disparou.
func (r *Repository) ListRuns(ctx context.Context, enterpriseID int64, limit int) ([]planning_uc.RunRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT h.id, h.enterprise_id, h.plan_code, h.trigger, h.triggered_by, COALESCE(u.name, ''),
		       h.snapshot_at, h.started_at, h.finished_at, h.status,
		       h.changed_items, h.mrp_orders, h.crp_overload, h.viable, COALESCE(h.notes, '')
		  FROM planning_run_history h
		  LEFT JOIN users u ON u.id = h.triggered_by
		 WHERE h.enterprise_id = $1
		 ORDER BY h.started_at DESC, h.id DESC
		 LIMIT $2`, enterpriseID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []planning_uc.RunRecord{}
	for rows.Next() {
		var rec planning_uc.RunRecord
		var trigger string
		if err := rows.Scan(&rec.ID, &rec.EnterpriseID, &rec.PlanCode, &trigger, &rec.TriggeredBy, &rec.TriggeredByName,
			&rec.SnapshotAt, &rec.StartedAt, &rec.FinishedAt, &rec.Status,
			&rec.ChangedItems, &rec.MRPOrders, &rec.CRPOverload, &rec.Viable, &rec.Notes); err != nil {
			return nil, err
		}
		rec.Trigger = planning_uc.Trigger(trigger)
		out = append(out, rec)
	}
	return out, rows.Err()
}

// TryLock usa advisory lock de sessão: some sozinho se o processo cair, o que
// evita um cadeado órfão travando o planejamento até alguém intervir.
//
// O cadeado pertence à *conexão* que o tomou, e o pool não garante devolver a
// mesma conexão na chamada seguinte. Por isso a conexão é retirada do pool e
// guardada até o Unlock — sem isso o unlock rodaria em outra sessão, não teria
// efeito, e a empresa ficaria travada até o processo reiniciar.
func (r *Repository) TryLock(ctx context.Context, enterpriseID int64) (bool, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return false, err
	}
	var ok bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1, $2)`, lockNamespace, enterpriseID).Scan(&ok); err != nil {
		conn.Release()
		return false, err
	}
	if !ok {
		conn.Release()
		return false, nil
	}
	r.mu.Lock()
	r.locks[enterpriseID] = conn
	r.mu.Unlock()
	return true, nil
}

func (r *Repository) Unlock(ctx context.Context, enterpriseID int64) error {
	r.mu.Lock()
	conn := r.locks[enterpriseID]
	delete(r.locks, enterpriseID)
	r.mu.Unlock()
	if conn == nil {
		return nil
	}
	defer conn.Release()
	_, err := conn.Exec(ctx, `SELECT pg_advisory_unlock($1, $2)`, lockNamespace, enterpriseID)
	return err
}

// ChangedItemsSince conta os itens tocados depois do corte anterior. É o que
// permite dizer, no histórico, quanta coisa mudou entre um ciclo e outro.
//
// O cadastro de item não tem updated_at: conta-se o item criado depois do
// corte. Antes a consulta lia i.updated_at e, da segunda rodada em diante
// (quando já existe corte), toda execução falhava com coluna inexistente.
func (r *Repository) ChangedItemsSince(ctx context.Context, enterpriseID int64, since time.Time) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT code) FROM (
		    SELECT i.code            FROM items i            WHERE i.enterprise_id = $1 AND i.created_at > $2
		    UNION
		    SELECT s.parent_code     FROM item_structures s
		      JOIN items p ON p.code = s.parent_code AND p.enterprise_id = $1
		     WHERE s.updated_at > $2
		    UNION
		    SELECT b.item_code       FROM stock_balances b   WHERE b.enterprise_id = $1 AND b.updated_at > $2
		    UNION
		    SELECT o.item_code       FROM production_orders o WHERE o.enterprise_id = $1 AND o.updated_at > $2
		) AS mudou(code)`, enterpriseID, since).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (r *Repository) StartRun(ctx context.Context, rec planning_uc.RunRecord) (int64, error) {
	var id int64
	var by *uuid.UUID = rec.TriggeredBy
	err := r.pool.QueryRow(ctx, `
		INSERT INTO planning_run_history
		    (enterprise_id, plan_code, trigger, triggered_by, snapshot_at, changed_items, status)
		VALUES ($1,$2,$3,$4,$5,$6,'RUNNING') RETURNING id`,
		rec.EnterpriseID, rec.PlanCode, string(rec.Trigger), by, rec.SnapshotAt, rec.ChangedItems).Scan(&id)
	return id, err
}

func (r *Repository) FinishRun(ctx context.Context, id int64, rec planning_uc.RunRecord) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE planning_run_history
		   SET finished_at = now(), status = $2, mrp_orders = $3,
		       crp_overload = $4, viable = $5, notes = $6
		 WHERE id = $1`,
		id, rec.Status, rec.MRPOrders, rec.CRPOverload, rec.Viable, rec.Notes)
	return err
}

func (r *Repository) SaveSnapshot(ctx context.Context, enterpriseID int64, snapshotAt time.Time, status string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO planning_auto_run_settings (enterprise_id, last_snapshot_at, last_run_at, last_run_status)
		VALUES ($1, $2, now(), $3)
		ON CONFLICT (enterprise_id) DO UPDATE
		   SET last_snapshot_at = CASE WHEN $3 = 'SUCCESS' THEN EXCLUDED.last_snapshot_at
		                               ELSE planning_auto_run_settings.last_snapshot_at END,
		       last_run_at      = EXCLUDED.last_run_at,
		       last_run_status  = EXCLUDED.last_run_status,
		       updated_at       = now()`, enterpriseID, snapshotAt, status)
	return err
}
