package recurring_sales

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryPGX struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *RepositoryPGX { return &RepositoryPGX{pool: pool} }

func (r *RepositoryPGX) UpsertParameters(ctx context.Context, p *entity.Parameters) (*entity.Parameters, error) {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	p.EnterpriseCode = enterpriseCode
	row := r.pool.QueryRow(ctx, `INSERT INTO recurring_sales_parameters
		(enterprise_code, current_month_billing_limit_day, group_order_item_total, indefinite_delivery_day,
		 fixed_term_delivery_day, consider_discounts_additions, generic_representative_code, generic_sales_plan_code, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (enterprise_code) DO UPDATE SET
			current_month_billing_limit_day=EXCLUDED.current_month_billing_limit_day,
			group_order_item_total=EXCLUDED.group_order_item_total,
			indefinite_delivery_day=EXCLUDED.indefinite_delivery_day,
			fixed_term_delivery_day=EXCLUDED.fixed_term_delivery_day,
			consider_discounts_additions=EXCLUDED.consider_discounts_additions,
			generic_representative_code=EXCLUDED.generic_representative_code,
			generic_sales_plan_code=EXCLUDED.generic_sales_plan_code,
			updated_at=NOW(), updated_by=EXCLUDED.updated_by
		RETURNING enterprise_code, current_month_billing_limit_day, group_order_item_total, indefinite_delivery_day,
		          fixed_term_delivery_day, consider_discounts_additions, generic_representative_code, generic_sales_plan_code,
		          updated_at, updated_by`,
		p.EnterpriseCode, p.CurrentMonthBillingLimitDay, p.GroupOrderItemTotal, p.IndefiniteDeliveryDay,
		p.FixedTermDeliveryDay, p.ConsiderDiscountsAdditions, p.GenericRepresentativeCode, p.GenericSalesPlanCode,
		pgutil.ToPgUUID(p.UpdatedBy))
	return scanParameters(row)
}

func (r *RepositoryPGX) GetParameters(ctx context.Context, enterpriseCode int64) (*entity.Parameters, error) {
	authenticatedCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	enterpriseCode = authenticatedCode
	row := r.pool.QueryRow(ctx, `SELECT enterprise_code, current_month_billing_limit_day, group_order_item_total,
		indefinite_delivery_day, fixed_term_delivery_day, consider_discounts_additions, generic_representative_code,
		generic_sales_plan_code, updated_at, updated_by FROM recurring_sales_parameters WHERE enterprise_code=$1`, enterpriseCode)
	return scanParameters(row)
}

func (r *RepositoryPGX) CreateAdjustmentDate(ctx context.Context, v *entity.AdjustmentDate) (*entity.AdjustmentDate, error) {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	v.EnterpriseCode = enterpriseCode
	row := r.pool.QueryRow(ctx, `INSERT INTO recurring_sales_adjustment_dates
		(enterprise_code, customer_code, establishment_code, adjustment_date, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING code, enterprise_code, customer_code, establishment_code, adjustment_date, notes, created_at, created_by`,
		v.EnterpriseCode, v.CustomerCode, v.EstablishmentCode, v.AdjustmentDate, v.Notes, pgutil.ToPgUUID(v.CreatedBy))
	return scanAdjustmentDate(row)
}

func (r *RepositoryPGX) ListAdjustmentDates(ctx context.Context, filter repository.Filter) ([]*entity.AdjustmentDate, error) {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	filter.EnterpriseCode = &enterpriseCode
	conds, args := buildFilter(filter, "d", false)
	q := `SELECT d.code, d.enterprise_code, d.customer_code, d.establishment_code, d.adjustment_date, d.notes, d.created_at, d.created_by FROM recurring_sales_adjustment_dates d`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY d.adjustment_date DESC LIMIT 500"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.AdjustmentDate
	for rows.Next() {
		row, err := scanAdjustmentDate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) Create(ctx context.Context, v *entity.RecurringSale) (*entity.RecurringSale, error) {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	v.EnterpriseCode = enterpriseCode
	row := r.pool.QueryRow(ctx, recurringInsertSQL(), recurringArgs(v)...)
	created, err := scanRecurringSale(row)
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, created)
}

func (r *RepositoryPGX) CancelAtomic(ctx context.Context, command repository.CancellationCommand) (*entity.RecurringSale, error) {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var beforeJSON []byte
	var status string
	err = tx.QueryRow(ctx, `SELECT to_jsonb(r), r.lifecycle_status FROM recurring_sales r
		WHERE r.code=$1 AND r.enterprise_code=$2 FOR UPDATE`, command.Code, enterpriseCode).Scan(&beforeJSON, &status)
	if err != nil {
		return nil, err
	}
	if status != string(entity.LifecycleActive) && status != string(entity.LifecycleSuspended) {
		return nil, repository.ErrInvalidLifecycleState
	}

	row := tx.QueryRow(ctx, `UPDATE recurring_sales SET lifecycle_status='CANCELADA', is_active=FALSE,
		cancellation_effective_date=$3, effective_until=$3, future_orders_policy=$4,
		cancelled_by=$5, reason=$6, updated_at=NOW()
		WHERE code=$1 AND enterprise_code=$2 RETURNING `+recurringColumns(), command.Code, enterpriseCode,
		command.EffectiveDate, command.FutureOrdersPolicy, pgutil.ToPgUUID(command.ActorID), command.Reason)
	updated, err := scanRecurringSale(row)
	if err != nil {
		return nil, err
	}
	var afterJSON []byte
	if err = tx.QueryRow(ctx, `SELECT to_jsonb(r) FROM recurring_sales r WHERE code=$1 AND enterprise_code=$2`, command.Code, enterpriseCode).Scan(&afterJSON); err != nil {
		return nil, err
	}
	if !json.Valid(beforeJSON) || !json.Valid(afterJSON) {
		return nil, fmt.Errorf("não foi possível gerar a auditoria da recorrência")
	}
	if _, err = tx.Exec(ctx, `INSERT INTO recurring_sales_events
		(enterprise_code,recurring_sale_code,event_type,before_state,after_state,reason,correlation_id,actor_id)
		VALUES($1,$2,'CANCELADA',$3,$4,$5,NULLIF($6,''),$7)`, enterpriseCode, command.Code,
		beforeJSON, afterJSON, command.Reason, command.CorrelationID, pgutil.ToPgUUID(command.ActorID)); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.hydrate(ctx, updated)
}

func (r *RepositoryPGX) ReserveOperation(ctx context.Context, operation *entity.Operation) (*entity.Operation, bool, error) {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, false, err
	}
	tag, err := r.pool.Exec(ctx, `INSERT INTO recurring_sales_operations(enterprise_code,recurring_sale_code,operation_type,competence,idempotency_key,request_hash,actor_id)
		SELECT $1,$2,$3,$4,$5,$6,$7 WHERE EXISTS(SELECT 1 FROM recurring_sales WHERE code=$2 AND enterprise_code=$1)
		ON CONFLICT DO NOTHING`, enterpriseCode, operation.RecurringSaleCode, operation.OperationType, operation.Competence,
		operation.IdempotencyKey, operation.RequestHash, pgutil.ToPgUUID(operation.ActorID))
	if err != nil {
		return nil, false, err
	}
	if tag.RowsAffected() == 1 {
		operation.Status = "EM_PROCESSAMENTO"
		return operation, false, nil
	}
	existing := &entity.Operation{}
	err = r.pool.QueryRow(ctx, `SELECT recurring_sale_code,operation_type,competence,idempotency_key,request_hash,result_code,status,actor_id
		FROM recurring_sales_operations WHERE enterprise_code=$1 AND (idempotency_key=$2 OR (recurring_sale_code=$3 AND operation_type=$4 AND competence=$5))`,
		enterpriseCode, operation.IdempotencyKey, operation.RecurringSaleCode, operation.OperationType, operation.Competence).
		Scan(&existing.RecurringSaleCode, &existing.OperationType, &existing.Competence, &existing.IdempotencyKey, &existing.RequestHash, &existing.ResultCode, &existing.Status, &existing.ActorID)
	if err != nil {
		return nil, false, err
	}
	if existing.RecurringSaleCode != operation.RecurringSaleCode || existing.OperationType != operation.OperationType || existing.Competence != operation.Competence || existing.RequestHash != operation.RequestHash {
		return nil, false, repository.ErrOperationConflict
	}
	if existing.Status == "CONCLUIDA" {
		return existing, true, nil
	}
	if existing.Status == "EM_PROCESSAMENTO" {
		return nil, false, repository.ErrOperationInProgress
	}
	_, err = r.pool.Exec(ctx, `UPDATE recurring_sales_operations SET status='EM_PROCESSAMENTO',actor_id=$3,created_at=NOW(),completed_at=NULL WHERE enterprise_code=$1 AND idempotency_key=$2`, enterpriseCode, operation.IdempotencyKey, pgutil.ToPgUUID(operation.ActorID))
	return operation, false, err
}

func (r *RepositoryPGX) CompleteOperation(ctx context.Context, operation *entity.Operation, resultCode int64) error {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `UPDATE recurring_sales_operations SET status='CONCLUIDA',result_code=$3,completed_at=NOW() WHERE enterprise_code=$1 AND idempotency_key=$2 AND status='EM_PROCESSAMENTO'`, enterpriseCode, operation.IdempotencyKey, resultCode)
	return err
}

func (r *RepositoryPGX) FailOperation(ctx context.Context, operation *entity.Operation) error {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `UPDATE recurring_sales_operations SET status='FALHOU',completed_at=NOW() WHERE enterprise_code=$1 AND idempotency_key=$2 AND status='EM_PROCESSAMENTO'`, enterpriseCode, operation.IdempotencyKey)
	return err
}

func (r *RepositoryPGX) CreateWithRepresentatives(ctx context.Context, v *entity.RecurringSale) (*entity.RecurringSale, error) {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	v.EnterpriseCode = enterpriseCode
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	created, err := scanRecurringSale(tx.QueryRow(ctx, recurringInsertSQL(), recurringArgs(v)...))
	if err != nil {
		return nil, err
	}
	for _, representative := range v.Representatives {
		_, err = tx.Exec(ctx, `INSERT INTO recurring_sales_representatives (recurring_sale_code,representative_code,is_primary,commission_percent,commission_base,is_lifetime,commission_installments) VALUES($1,$2,$3,$4,$5,$6,$7)`, created.Code, representative.RepresentativeCode, representative.IsPrimary, representative.CommissionPercent, string(representative.CommissionBase), representative.IsLifetime, representative.CommissionInstallments)
		if err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.Get(ctx, created.Code)
}

func (r *RepositoryPGX) Update(ctx context.Context, v *entity.RecurringSale) (*entity.RecurringSale, error) {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `UPDATE recurring_sales SET
		sales_plan_code=$2, sale_date=$3, next_adjustment_date=$4, months_quantity=$5, payments_quantity=$6,
		grace_months=$7, payment_value=$8, quantity=$9, unit_value=$10, reason=$11, adjustment_percent=$12,
		is_active=$13, updated_at=NOW()
		WHERE code=$1 AND enterprise_code=$14
		RETURNING `+recurringColumns(),
		v.Code, v.SalesPlanCode, v.SaleDate, v.NextAdjustmentDate, v.MonthsQuantity, v.PaymentsQuantity,
		v.GraceMonths, v.PaymentValue, v.Quantity, v.UnitValue, v.Reason, v.AdjustmentPercent, v.IsActive, enterpriseCode)
	updated, err := scanRecurringSale(row)
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, updated)
}

func (r *RepositoryPGX) Get(ctx context.Context, code int64) (*entity.RecurringSale, error) {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `SELECT `+recurringColumns()+` FROM recurring_sales WHERE code=$1 AND enterprise_code=$2`, code, enterpriseCode)
	rec, err := scanRecurringSale(row)
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, rec)
}

func (r *RepositoryPGX) List(ctx context.Context, filter repository.Filter) ([]*entity.RecurringSale, error) {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	filter.EnterpriseCode = &enterpriseCode
	conds, args := buildFilter(filter, "r", true)
	q := `SELECT ` + recurringColumnsWithAlias("r") + ` FROM recurring_sales r`
	if filter.RepresentativeCode != nil {
		q += ` JOIN recurring_sales_representatives rr ON rr.recurring_sale_code=r.code`
	}
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY r.sale_date DESC, r.code DESC LIMIT 1000"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.RecurringSale
	for rows.Next() {
		row, err := scanRecurringSale(rows)
		if err != nil {
			return nil, err
		}
		hydrated, err := r.hydrate(ctx, row)
		if err != nil {
			return nil, err
		}
		out = append(out, hydrated)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) AddRepresentative(ctx context.Context, v *entity.Representative) (*entity.Representative, error) {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `INSERT INTO recurring_sales_representatives
		(recurring_sale_code, representative_code, is_primary, commission_percent, commission_base, is_lifetime, commission_installments)
		SELECT $1,$2,$3,$4,$5,$6,$7 WHERE EXISTS (SELECT 1 FROM recurring_sales WHERE code=$1 AND enterprise_code=$8)
		RETURNING code, recurring_sale_code, representative_code, is_primary, commission_percent, commission_base, is_lifetime, commission_installments, created_at`,
		v.RecurringSaleCode, v.RepresentativeCode, v.IsPrimary, v.CommissionPercent, string(v.CommissionBase), v.IsLifetime, v.CommissionInstallments, enterpriseCode)
	return scanRepresentative(row)
}

func (r *RepositoryPGX) MarkOrderGenerated(ctx context.Context, code int64, orderCode int64) (*entity.RecurringSale, error) {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `UPDATE recurring_sales SET generated_order_code=$2, generated_order_at=NOW(), updated_at=NOW()
		WHERE code=$1 AND enterprise_code=$3 AND generated_order_code IS NULL AND movement_type IN ('SALE','UPGRADE','ADJUSTMENT')
		RETURNING `+recurringColumns(), code, orderCode, enterpriseCode)
	rec, err := scanRecurringSale(row)
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, rec)
}

func (r *RepositoryPGX) ClearGeneratedOrder(ctx context.Context, code int64) (*entity.RecurringSale, error) {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `UPDATE recurring_sales SET generated_order_code=NULL, generated_order_at=NULL, updated_at=NOW()
		WHERE code=$1 AND enterprise_code=$2
		RETURNING `+recurringColumns(), code, enterpriseCode)
	rec, err := scanRecurringSale(row)
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, rec)
}

func (r *RepositoryPGX) Deactivate(ctx context.Context, code int64, reason *string) (*entity.RecurringSale, error) {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `UPDATE recurring_sales SET is_active=FALSE, reason=COALESCE($2, reason), updated_at=NOW()
		WHERE code=$1 AND enterprise_code=$3 RETURNING `+recurringColumns(), code, reason, enterpriseCode)
	rec, err := scanRecurringSale(row)
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, rec)
}

func (r *RepositoryPGX) CreateAdjustmentLink(ctx context.Context, adjustmentCode, sourceCode int64) error {
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO recurring_sales_adjustment_links (adjustment_code, source_recurring_sale_code)
		SELECT $1,$2 WHERE EXISTS (SELECT 1 FROM recurring_sales a JOIN recurring_sales s ON s.code=$2 AND s.enterprise_code=$3 WHERE a.code=$1 AND a.enterprise_code=$3)
		ON CONFLICT DO NOTHING`, adjustmentCode, sourceCode, enterpriseCode)
	return err
}

func (r *RepositoryPGX) hydrate(ctx context.Context, rec *entity.RecurringSale) (*entity.RecurringSale, error) {
	var lifecycle string
	var cancelledBy pgtype.UUID
	if err := r.pool.QueryRow(ctx, `SELECT lifecycle_status,effective_from,effective_until,cancellation_effective_date,
		future_orders_policy,cancelled_by,frequency,price_table_code,currency_code,adjustment_index,adjustment_period_months,
		adjustment_floor_pct,adjustment_cap_pct,billing_policy,delivery_policy,tax_policy,cost_center_code,renewal_policy
		FROM recurring_sales WHERE code=$1 AND enterprise_code=$2`, rec.Code, rec.EnterpriseCode).
		Scan(&lifecycle, &rec.EffectiveFrom, &rec.EffectiveUntil, &rec.CancellationEffectiveDate, &rec.FutureOrdersPolicy, &cancelledBy,
			&rec.Frequency, &rec.PriceTableCode, &rec.CurrencyCode, &rec.AdjustmentIndex, &rec.AdjustmentPeriodMonths,
			&rec.AdjustmentFloorPct, &rec.AdjustmentCapPct, &rec.BillingPolicy, &rec.DeliveryPolicy, &rec.TaxPolicy,
			&rec.CostCenterCode, &rec.RenewalPolicy); err != nil {
		return nil, err
	}
	rec.LifecycleStatus = entity.LifecycleStatus(lifecycle)
	if cancelledBy.Valid {
		actor := uuid.UUID(cancelledBy.Bytes)
		rec.CancelledBy = &actor
	}
	rows, err := r.pool.Query(ctx, `SELECT code, recurring_sale_code, representative_code, is_primary, commission_percent,
		commission_base, is_lifetime, commission_installments, created_at
		FROM recurring_sales_representatives rr WHERE recurring_sale_code=$1 AND EXISTS (SELECT 1 FROM recurring_sales r WHERE r.code=rr.recurring_sale_code AND r.enterprise_code=$2) ORDER BY is_primary DESC, representative_code`, rec.Code, rec.EnterpriseCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		rep, err := scanRepresentative(rows)
		if err != nil {
			return nil, err
		}
		rec.Representatives = append(rec.Representatives, rep)
	}
	return rec, rows.Err()
}

func buildFilter(filter repository.Filter, alias string, recurring bool) ([]string, []any) {
	prefix := alias + "."
	args := []any{}
	conds := []string{}
	add := func(col string, v any) {
		args = append(args, v)
		conds = append(conds, fmt.Sprintf("%s%s=$%d", prefix, col, len(args)))
	}
	if filter.EnterpriseCode != nil {
		add("enterprise_code", *filter.EnterpriseCode)
	}
	if filter.CustomerCode != nil {
		add("customer_code", *filter.CustomerCode)
	}
	if filter.EstablishmentCode != nil {
		add("establishment_code", *filter.EstablishmentCode)
	}
	if recurring {
		if filter.ItemCode != nil {
			add("item_code", *filter.ItemCode)
		}
		if filter.MovementType != nil {
			add("movement_type", string(*filter.MovementType))
		}
		if filter.OnlyActive {
			conds = append(conds, prefix+"is_active")
		}
		if filter.RepresentativeCode != nil {
			args = append(args, *filter.RepresentativeCode)
			conds = append(conds, fmt.Sprintf("rr.representative_code=$%d", len(args)))
		}
	}
	return conds, args
}

func recurringInsertSQL() string {
	return `INSERT INTO recurring_sales
		(enterprise_code, customer_code, establishment_code, item_code, item_mask, sales_plan_code, movement_type, term_type,
		 sale_date, next_adjustment_date, months_quantity, payments_quantity, grace_months, payment_value, quantity,
		 unit_value, reason, generated_order_code, generated_order_at, source_recurring_sale_code, original_adjustment_code,
		 adjustment_percent, is_active, lifecycle_status, effective_from, effective_until, frequency, price_table_code,
		 currency_code, adjustment_index, adjustment_period_months, adjustment_floor_pct, adjustment_cap_pct,
		 billing_policy, delivery_policy, tax_policy, cost_center_code, renewal_policy, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39)
		RETURNING ` + recurringColumns()
}

func recurringArgs(v *entity.RecurringSale) []any {
	return []any{
		v.EnterpriseCode, v.CustomerCode, v.EstablishmentCode, v.ItemCode, v.ItemMask, v.SalesPlanCode,
		string(v.MovementType), string(v.TermType), v.SaleDate, v.NextAdjustmentDate, v.MonthsQuantity,
		v.PaymentsQuantity, v.GraceMonths, v.PaymentValue, v.Quantity, v.UnitValue, v.Reason,
		v.GeneratedOrderCode, v.GeneratedOrderAt, v.SourceRecurringSaleCode, v.OriginalAdjustmentCode,
		v.AdjustmentPercent, v.IsActive, string(v.LifecycleStatus), v.EffectiveFrom, v.EffectiveUntil, v.Frequency,
		v.PriceTableCode, v.CurrencyCode, v.AdjustmentIndex, v.AdjustmentPeriodMonths, v.AdjustmentFloorPct,
		v.AdjustmentCapPct, v.BillingPolicy, v.DeliveryPolicy, v.TaxPolicy, v.CostCenterCode, v.RenewalPolicy,
		pgutil.ToPgUUID(v.CreatedBy),
	}
}

func recurringColumns() string {
	return `code, enterprise_code, customer_code, establishment_code, item_code, item_mask, sales_plan_code, movement_type,
		term_type, sale_date, next_adjustment_date, months_quantity, payments_quantity, grace_months, payment_value,
		quantity, unit_value, reason, generated_order_code, generated_order_at, source_recurring_sale_code,
		original_adjustment_code, adjustment_percent, is_active, created_at, updated_at, created_by`
}

func recurringColumnsWithAlias(alias string) string {
	cols := strings.Split(recurringColumns(), ",")
	for i, col := range cols {
		cols[i] = alias + "." + strings.TrimSpace(col)
	}
	return strings.Join(cols, ", ")
}

func scanParameters(row pgx.Row) (*entity.Parameters, error) {
	var v entity.Parameters
	err := row.Scan(&v.EnterpriseCode, &v.CurrentMonthBillingLimitDay, &v.GroupOrderItemTotal, &v.IndefiniteDeliveryDay,
		&v.FixedTermDeliveryDay, &v.ConsiderDiscountsAdditions, &v.GenericRepresentativeCode, &v.GenericSalesPlanCode,
		&v.UpdatedAt, &v.UpdatedBy)
	return &v, err
}

func scanAdjustmentDate(row pgx.Row) (*entity.AdjustmentDate, error) {
	var v entity.AdjustmentDate
	err := row.Scan(&v.Code, &v.EnterpriseCode, &v.CustomerCode, &v.EstablishmentCode, &v.AdjustmentDate,
		&v.Notes, &v.CreatedAt, &v.CreatedBy)
	return &v, err
}

func scanRecurringSale(row pgx.Row) (*entity.RecurringSale, error) {
	var v entity.RecurringSale
	err := row.Scan(&v.Code, &v.EnterpriseCode, &v.CustomerCode, &v.EstablishmentCode, &v.ItemCode, &v.ItemMask,
		&v.SalesPlanCode, &v.MovementType, &v.TermType, &v.SaleDate, &v.NextAdjustmentDate, &v.MonthsQuantity,
		&v.PaymentsQuantity, &v.GraceMonths, &v.PaymentValue, &v.Quantity, &v.UnitValue, &v.Reason,
		&v.GeneratedOrderCode, &v.GeneratedOrderAt, &v.SourceRecurringSaleCode, &v.OriginalAdjustmentCode,
		&v.AdjustmentPercent, &v.IsActive, &v.CreatedAt, &v.UpdatedAt, &v.CreatedBy)
	return &v, err
}

func scanRepresentative(row pgx.Row) (*entity.Representative, error) {
	var v entity.Representative
	err := row.Scan(&v.Code, &v.RecurringSaleCode, &v.RepresentativeCode, &v.IsPrimary, &v.CommissionPercent,
		&v.CommissionBase, &v.IsLifetime, &v.CommissionInstallments, &v.CreatedAt)
	return &v, err
}

var _ repository.Repository = (*RepositoryPGX)(nil)
