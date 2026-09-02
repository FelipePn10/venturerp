package commercial_commission

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/commercial_commission/entity"
	commissionrepo "github.com/FelipePn10/panossoerp/internal/domain/commercial_commission/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }
func (r *Repository) UpsertSettings(ctx context.Context, tenant int64, v *entity.Settings) (*entity.Settings, error) {
	err := r.pool.QueryRow(ctx, `INSERT INTO commercial_commission_settings(enterprise_id,competence_event,invoice_share_pct,receipt_share_pct,updated_by) VALUES($1,$2,$3,$4,$5) ON CONFLICT(enterprise_id) DO UPDATE SET competence_event=EXCLUDED.competence_event,invoice_share_pct=EXCLUDED.invoice_share_pct,receipt_share_pct=EXCLUDED.receipt_share_pct,updated_by=EXCLUDED.updated_by,updated_at=NOW() RETURNING enterprise_id,competence_event,invoice_share_pct::text,receipt_share_pct::text,updated_at,updated_by`, tenant, v.CompetenceEvent, v.InvoiceSharePct, v.ReceiptSharePct, pgutil.ToPgUUID(v.UpdatedBy)).Scan(&v.EnterpriseID, &v.CompetenceEvent, &v.InvoiceSharePct, &v.ReceiptSharePct, &v.UpdatedAt, &v.UpdatedBy)
	return v, err
}
func (r *Repository) GetSettings(ctx context.Context, tenant int64) (*entity.Settings, error) {
	v := &entity.Settings{}
	err := r.pool.QueryRow(ctx, `SELECT enterprise_id,competence_event,invoice_share_pct::text,receipt_share_pct::text,updated_at,updated_by FROM commercial_commission_settings WHERE enterprise_id=$1`, tenant).Scan(&v.EnterpriseID, &v.CompetenceEvent, &v.InvoiceSharePct, &v.ReceiptSharePct, &v.UpdatedAt, &v.UpdatedBy)
	return v, err
}
func (r *Repository) List(ctx context.Context, tenant int64, f entity.Filter) ([]*entity.LedgerEntry, error) {
	conds := []string{"enterprise_id=$1"}
	args := []any{tenant}
	add := func(c string, v any) { args = append(args, v); conds = append(conds, fmt.Sprintf(c, len(args))) }
	if f.RepresentativeCode != nil {
		add("representative_code=$%d", *f.RepresentativeCode)
	}
	if f.Status != nil {
		add("status=$%d", *f.Status)
	}
	if f.From != nil {
		add("competence_date >= $%d", *f.From)
	}
	if f.To != nil {
		add("competence_date <= $%d", *f.To)
	}
	args = append(args, f.Limit, f.Offset)
	q := `SELECT code,representative_code,sales_order_code,fiscal_exit_id,receivable_id,event_type,competence_date,base_amount::text,commission_pct::text,amount::text,status,reversal_of,occurred_at,reconciled_at,paid_at,payment_reference FROM commercial_commission_ledger WHERE ` + strings.Join(conds, " AND ") + fmt.Sprintf(" ORDER BY competence_date DESC,code DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*entity.LedgerEntry, 0)
	for rows.Next() {
		v := &entity.LedgerEntry{}
		if err = scanEntry(rows, v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *Repository) Transition(ctx context.Context, tenant int64, c entity.TransitionCommand) (*entity.LedgerEntry, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var ledger int64
	var typ string
	err = tx.QueryRow(ctx, `SELECT ledger_code,event_type FROM commercial_commission_events WHERE enterprise_id=$1 AND idempotency_key=$2`, tenant, c.IdempotencyKey).Scan(&ledger, &typ)
	if err == nil {
		if ledger != c.Code || typ != c.Action {
			return nil, commissionrepo.ErrIdempotencyConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, err
		}
		return r.get(ctx, tenant, c.Code)
	}
	if err != pgx.ErrNoRows {
		return nil, err
	}
	current := &entity.LedgerEntry{}
	if err = scanEntry(tx.QueryRow(ctx, entrySelect+` WHERE enterprise_id=$1 AND code=$2 FOR UPDATE`, tenant, c.Code), current); err != nil {
		return nil, err
	}
	next := map[string]string{"CONCILIADA": "CONCILIADO", "PAGA": "PAGO"}[c.Action]
	if next == "" || c.Action == "CONCILIADA" && current.Status != "ABERTO" || c.Action == "PAGA" && current.Status != "CONCILIADO" {
		return nil, commissionrepo.ErrInvalidState
	}
	if c.Action == "PAGA" && (c.PaymentReference == nil || strings.TrimSpace(*c.PaymentReference) == "") {
		return nil, commissionrepo.ErrInvalidState
	}
	before, _ := json.Marshal(current)
	updated := &entity.LedgerEntry{}
	err = scanEntry(tx.QueryRow(ctx, `UPDATE commercial_commission_ledger SET status=$3::varchar,reconciled_at=CASE WHEN $3::varchar='CONCILIADO' THEN NOW() ELSE reconciled_at END,reconciled_by=CASE WHEN $3::varchar='CONCILIADO' THEN $4::uuid ELSE reconciled_by END,paid_at=CASE WHEN $3::varchar='PAGO' THEN NOW() ELSE paid_at END,paid_by=CASE WHEN $3::varchar='PAGO' THEN $4::uuid ELSE paid_by END,payment_reference=CASE WHEN $3::varchar='PAGO' THEN $5::varchar ELSE payment_reference END WHERE enterprise_id=$1 AND code=$2 RETURNING `+entryColumns, tenant, c.Code, next, pgutil.ToPgUUID(c.ActorID), c.PaymentReference), updated)
	if err != nil {
		return nil, err
	}
	after, _ := json.Marshal(updated)
	_, err = tx.Exec(ctx, `INSERT INTO commercial_commission_events(enterprise_id,ledger_code,event_type,before_state,after_state,reason,idempotency_key,actor_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, tenant, c.Code, c.Action, before, after, c.Reason, c.IdempotencyKey, pgutil.ToPgUUID(c.ActorID))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

const entryColumns = `code,representative_code,sales_order_code,fiscal_exit_id,receivable_id,event_type,competence_date,base_amount::text,commission_pct::text,amount::text,status,reversal_of,occurred_at,reconciled_at,paid_at,payment_reference`
const entrySelect = `SELECT ` + entryColumns + ` FROM commercial_commission_ledger`

func (r *Repository) get(ctx context.Context, tenant, code int64) (*entity.LedgerEntry, error) {
	v := &entity.LedgerEntry{}
	err := scanEntry(r.pool.QueryRow(ctx, entrySelect+` WHERE enterprise_id=$1 AND code=$2`, tenant, code), v)
	return v, err
}

type scanner interface{ Scan(...any) error }

func scanEntry(s scanner, v *entity.LedgerEntry) error {
	return s.Scan(&v.Code, &v.RepresentativeCode, &v.SalesOrderCode, &v.FiscalExitID, &v.ReceivableID, &v.EventType, &v.CompetenceDate, &v.BaseAmount, &v.CommissionPct, &v.Amount, &v.Status, &v.ReversalOf, &v.OccurredAt, &v.ReconciledAt, &v.PaidAt, &v.PaymentReference)
}
