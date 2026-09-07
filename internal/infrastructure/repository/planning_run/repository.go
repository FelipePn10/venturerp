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

// DueSettings devolve as empresas cuja janela já passou hoje e que ainda não
// rodaram desde então. A comparação é feita no banco para não depender do fuso
// do processo.
func (r *Repository) DueSettings(ctx context.Context, now time.Time) ([]planning_uc.AutoRunSettings, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT enterprise_id, is_enabled, run_hour, run_minute,
		       COALESCE(plan_code, 0), initial_order_number, generate_llc,
		       last_snapshot_at, last_run_at
		  FROM planning_auto_run_settings
		 WHERE is_enabled
		   AND plan_code IS NOT NULL
		   AND make_timestamptz(
		         EXTRACT(YEAR FROM $1::timestamptz)::int,
		         EXTRACT(MONTH FROM $1::timestamptz)::int,
		         EXTRACT(DAY FROM $1::timestamptz)::int,
		         run_hour, run_minute, 0) <= $1
		   AND (last_run_at IS NULL OR last_run_at::date < $1::date)`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []planning_uc.AutoRunSettings{}
	for rows.Next() {
		var s planning_uc.AutoRunSettings
		if err := rows.Scan(&s.EnterpriseID, &s.IsEnabled, &s.RunHour, &s.RunMinute,
			&s.PlanCode, &s.InitialOrderNumber, &s.GenerateLLC,
			&s.LastSnapshotAt, &s.LastRunAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) Settings(ctx context.Context, enterpriseID int64) (*planning_uc.AutoRunSettings, error) {
	var s planning_uc.AutoRunSettings
	err := r.pool.QueryRow(ctx, `
		SELECT enterprise_id, is_enabled, run_hour, run_minute,
		       COALESCE(plan_code, 0), initial_order_number, generate_llc,
		       last_snapshot_at, last_run_at
		  FROM planning_auto_run_settings WHERE enterprise_id = $1`, enterpriseID).
		Scan(&s.EnterpriseID, &s.IsEnabled, &s.RunHour, &s.RunMinute,
			&s.PlanCode, &s.InitialOrderNumber, &s.GenerateLLC,
			&s.LastSnapshotAt, &s.LastRunAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &planning_uc.AutoRunSettings{EnterpriseID: enterpriseID, InitialOrderNumber: 1, GenerateLLC: true}, nil
		}
		return nil, err
	}
	return &s, nil
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
func (r *Repository) ChangedItemsSince(ctx context.Context, enterpriseID int64, since time.Time) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT code) FROM (
		    SELECT i.code            FROM items i            WHERE i.enterprise_id = $1 AND i.updated_at > $2
		    UNION
		    SELECT s.parent_code     FROM item_structures s
		      JOIN items p ON p.code = s.parent_code AND p.enterprise_id = $1
		     WHERE s.updated_at > $2
		    UNION
		    SELECT b.item_code       FROM stock_balances b   WHERE b.enterprise_id = $1 AND b.updated_at > $2
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
