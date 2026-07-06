package recurring_sales

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryPGX struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *RepositoryPGX { return &RepositoryPGX{pool: pool} }

func (r *RepositoryPGX) UpsertParameters(ctx context.Context, p *entity.Parameters) (*entity.Parameters, error) {
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
	row := r.pool.QueryRow(ctx, `SELECT enterprise_code, current_month_billing_limit_day, group_order_item_total,
		indefinite_delivery_day, fixed_term_delivery_day, consider_discounts_additions, generic_representative_code,
		generic_sales_plan_code, updated_at, updated_by FROM recurring_sales_parameters WHERE enterprise_code=$1`, enterpriseCode)
	return scanParameters(row)
}

func (r *RepositoryPGX) CreateAdjustmentDate(ctx context.Context, v *entity.AdjustmentDate) (*entity.AdjustmentDate, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO recurring_sales_adjustment_dates
		(enterprise_code, customer_code, establishment_code, adjustment_date, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING code, enterprise_code, customer_code, establishment_code, adjustment_date, notes, created_at, created_by`,
		v.EnterpriseCode, v.CustomerCode, v.EstablishmentCode, v.AdjustmentDate, v.Notes, pgutil.ToPgUUID(v.CreatedBy))
	return scanAdjustmentDate(row)
}

func (r *RepositoryPGX) ListAdjustmentDates(ctx context.Context, filter repository.Filter) ([]*entity.AdjustmentDate, error) {
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
	row := r.pool.QueryRow(ctx, recurringInsertSQL(), recurringArgs(v)...)
	return scanRecurringSale(row)
}

func (r *RepositoryPGX) Update(ctx context.Context, v *entity.RecurringSale) (*entity.RecurringSale, error) {
	row := r.pool.QueryRow(ctx, `UPDATE recurring_sales SET
		sales_plan_code=$2, sale_date=$3, next_adjustment_date=$4, months_quantity=$5, payments_quantity=$6,
		grace_months=$7, payment_value=$8, quantity=$9, unit_value=$10, reason=$11, adjustment_percent=$12,
		is_active=$13, updated_at=NOW()
		WHERE code=$1
		RETURNING `+recurringColumns(),
		v.Code, v.SalesPlanCode, v.SaleDate, v.NextAdjustmentDate, v.MonthsQuantity, v.PaymentsQuantity,
		v.GraceMonths, v.PaymentValue, v.Quantity, v.UnitValue, v.Reason, v.AdjustmentPercent, v.IsActive)
	updated, err := scanRecurringSale(row)
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, updated)
}

func (r *RepositoryPGX) Get(ctx context.Context, code int64) (*entity.RecurringSale, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+recurringColumns()+` FROM recurring_sales WHERE code=$1`, code)
	rec, err := scanRecurringSale(row)
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, rec)
}

func (r *RepositoryPGX) List(ctx context.Context, filter repository.Filter) ([]*entity.RecurringSale, error) {
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
	row := r.pool.QueryRow(ctx, `INSERT INTO recurring_sales_representatives
		(recurring_sale_code, representative_code, is_primary, commission_percent, commission_base, is_lifetime, commission_installments)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING code, recurring_sale_code, representative_code, is_primary, commission_percent, commission_base, is_lifetime, commission_installments, created_at`,
		v.RecurringSaleCode, v.RepresentativeCode, v.IsPrimary, v.CommissionPercent, string(v.CommissionBase), v.IsLifetime, v.CommissionInstallments)
	return scanRepresentative(row)
}

func (r *RepositoryPGX) MarkOrderGenerated(ctx context.Context, code int64, orderCode int64) (*entity.RecurringSale, error) {
	row := r.pool.QueryRow(ctx, `UPDATE recurring_sales SET generated_order_code=$2, generated_order_at=NOW(), updated_at=NOW()
		WHERE code=$1 AND movement_type IN ('SALE','UPGRADE','ADJUSTMENT')
		RETURNING `+recurringColumns(), code, orderCode)
	rec, err := scanRecurringSale(row)
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, rec)
}

func (r *RepositoryPGX) ClearGeneratedOrder(ctx context.Context, code int64) (*entity.RecurringSale, error) {
	row := r.pool.QueryRow(ctx, `UPDATE recurring_sales SET generated_order_code=NULL, generated_order_at=NULL, updated_at=NOW()
		WHERE code=$1
		RETURNING `+recurringColumns(), code)
	rec, err := scanRecurringSale(row)
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, rec)
}

func (r *RepositoryPGX) Deactivate(ctx context.Context, code int64, reason *string) (*entity.RecurringSale, error) {
	row := r.pool.QueryRow(ctx, `UPDATE recurring_sales SET is_active=FALSE, reason=COALESCE($2, reason), updated_at=NOW()
		WHERE code=$1 RETURNING `+recurringColumns(), code, reason)
	rec, err := scanRecurringSale(row)
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, rec)
}

func (r *RepositoryPGX) CreateAdjustmentLink(ctx context.Context, adjustmentCode, sourceCode int64) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO recurring_sales_adjustment_links (adjustment_code, source_recurring_sale_code)
		VALUES ($1,$2) ON CONFLICT DO NOTHING`, adjustmentCode, sourceCode)
	return err
}

func (r *RepositoryPGX) hydrate(ctx context.Context, rec *entity.RecurringSale) (*entity.RecurringSale, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, recurring_sale_code, representative_code, is_primary, commission_percent,
		commission_base, is_lifetime, commission_installments, created_at
		FROM recurring_sales_representatives WHERE recurring_sale_code=$1 ORDER BY is_primary DESC, representative_code`, rec.Code)
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
		 adjustment_percent, is_active, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)
		RETURNING ` + recurringColumns()
}

func recurringArgs(v *entity.RecurringSale) []any {
	return []any{
		v.EnterpriseCode, v.CustomerCode, v.EstablishmentCode, v.ItemCode, v.ItemMask, v.SalesPlanCode,
		string(v.MovementType), string(v.TermType), v.SaleDate, v.NextAdjustmentDate, v.MonthsQuantity,
		v.PaymentsQuantity, v.GraceMonths, v.PaymentValue, v.Quantity, v.UnitValue, v.Reason,
		v.GeneratedOrderCode, v.GeneratedOrderAt, v.SourceRecurringSaleCode, v.OriginalAdjustmentCode,
		v.AdjustmentPercent, v.IsActive, pgutil.ToPgUUID(v.CreatedBy),
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
