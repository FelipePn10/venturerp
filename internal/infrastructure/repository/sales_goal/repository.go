package sales_goal

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_goal/entity"
	goalrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_goal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreatePeriod(ctx context.Context, p *entity.Period) (*entity.Period, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO public.sales_goal_periods (description,period_type,start_date,end_date,is_active)
VALUES ($1,$2,$3,$4,$5)
RETURNING code,description,period_type,start_date,end_date,is_active,created_at,updated_at`, p.Description, p.PeriodType, p.StartDate, p.EndDate, p.IsActive)
	return scanPeriod(row)
}

func (r *Repository) ListPeriods(ctx context.Context, filter goalrepo.PeriodFilter) ([]*entity.Period, error) {
	sqlText := `SELECT code,description,period_type,start_date,end_date,is_active,created_at,updated_at FROM public.sales_goal_periods WHERE TRUE`
	args := []any{}
	add := func(clause string, arg any) {
		args = append(args, arg)
		sqlText += fmt.Sprintf(" AND "+clause, len(args))
	}
	if filter.From != nil {
		add("end_date >= $%d", *filter.From)
	}
	if filter.To != nil {
		add("start_date <= $%d", *filter.To)
	}
	if filter.OnlyActive {
		sqlText += " AND is_active=TRUE"
	}
	sqlText += " ORDER BY start_date, code"
	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.Period{}
	for rows.Next() {
		p, err := scanPeriod(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) GetPeriod(ctx context.Context, code int64) (*entity.Period, error) {
	row := r.pool.QueryRow(ctx, `SELECT code,description,period_type,start_date,end_date,is_active,created_at,updated_at FROM public.sales_goal_periods WHERE code=$1`, code)
	return scanPeriod(row)
}

func (r *Repository) CreateGoal(ctx context.Context, g *entity.Goal) (*entity.Goal, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO public.sales_goals (representative_code,period_code,analysis_base,award_pct,notes,is_active)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING code,representative_code,period_code,analysis_base,award_pct,notes,is_active,created_at,updated_at`, g.RepresentativeCode, g.PeriodCode, g.AnalysisBase, g.AwardPct, g.Notes, g.IsActive)
	return scanGoal(row)
}

func (r *Repository) UpdateGoal(ctx context.Context, g *entity.Goal) (*entity.Goal, error) {
	row := r.pool.QueryRow(ctx, `UPDATE public.sales_goals SET representative_code=$2,period_code=$3,analysis_base=$4,award_pct=$5,notes=$6,is_active=$7,updated_at=NOW()
WHERE code=$1
RETURNING code,representative_code,period_code,analysis_base,award_pct,notes,is_active,created_at,updated_at`, g.Code, g.RepresentativeCode, g.PeriodCode, g.AnalysisBase, g.AwardPct, g.Notes, g.IsActive)
	return scanGoal(row)
}

func (r *Repository) GetGoal(ctx context.Context, code int64) (*entity.Goal, error) {
	row := r.pool.QueryRow(ctx, `SELECT code,representative_code,period_code,analysis_base,award_pct,notes,is_active,created_at,updated_at FROM public.sales_goals WHERE code=$1`, code)
	g, err := scanGoal(row)
	if err != nil {
		return nil, err
	}
	g.Items, err = r.listItems(ctx, code)
	return g, err
}

func (r *Repository) ListGoals(ctx context.Context, filter goalrepo.GoalFilter) ([]*entity.Goal, error) {
	sqlText := `SELECT code,representative_code,period_code,analysis_base,award_pct,notes,is_active,created_at,updated_at FROM public.sales_goals WHERE TRUE`
	args := []any{}
	add := func(clause string, arg any) {
		args = append(args, arg)
		sqlText += fmt.Sprintf(" AND "+clause, len(args))
	}
	if filter.RepresentativeCode != nil {
		add("representative_code=$%d", *filter.RepresentativeCode)
	}
	if filter.PeriodCode != nil {
		add("period_code=$%d", *filter.PeriodCode)
	}
	if filter.AnalysisBase != "" {
		add("analysis_base=$%d", filter.AnalysisBase)
	}
	if filter.OnlyActive {
		sqlText += " AND is_active=TRUE"
	}
	sqlText += " ORDER BY period_code, representative_code, code"
	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.Goal{}
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *Repository) AddGoalItem(ctx context.Context, item *entity.GoalItem) (*entity.GoalItem, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO public.sales_goal_items (goal_code,target_type,item_code,item_classification_code,item_group_code,sales_uom,target_quantity,target_value,bonus_pct,is_active)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING id,goal_code,target_type,item_code,item_classification_code,item_group_code,sales_uom,target_quantity,target_value,bonus_pct,is_active,created_at,updated_at`,
		item.GoalCode, item.TargetType, item.ItemCode, item.ItemClassificationCode, item.ItemGroupCode, item.SalesUOM, item.TargetQuantity, item.TargetValue, item.BonusPct, item.IsActive)
	return scanItem(row)
}

func (r *Repository) UpsertGroupTarget(ctx context.Context, target *entity.GroupTarget) (*entity.GroupTarget, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO public.sales_goal_group_targets (period_code,commercial_group_code,goal_type,minimum_value,minimum_bonus_pct,probable_value,probable_bonus_pct,ideal_value,ideal_bonus_pct,is_active)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (period_code,commercial_group_code,goal_type) DO UPDATE SET minimum_value=$4,minimum_bonus_pct=$5,probable_value=$6,probable_bonus_pct=$7,ideal_value=$8,ideal_bonus_pct=$9,is_active=$10,updated_at=NOW()
RETURNING id,period_code,commercial_group_code,goal_type,minimum_value,minimum_bonus_pct,probable_value,probable_bonus_pct,ideal_value,ideal_bonus_pct,is_active,created_at,updated_at`,
		target.PeriodCode, target.CommercialGroupCode, target.GoalType, target.MinimumValue, target.MinimumBonusPct, target.ProbableValue, target.ProbableBonusPct, target.IdealValue, target.IdealBonusPct, target.IsActive)
	return scanGroupTarget(row)
}

func (r *Repository) AddGroupCustomer(ctx context.Context, customer *entity.GroupCustomer) (*entity.GroupCustomer, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO public.sales_goal_group_customers (group_goal_id,customer_code,representative_code,minimum_value,minimum_bonus_pct,probable_value,probable_bonus_pct,ideal_value,ideal_bonus_pct,is_active)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (group_goal_id,customer_code) DO UPDATE SET representative_code=$3,minimum_value=$4,minimum_bonus_pct=$5,probable_value=$6,probable_bonus_pct=$7,ideal_value=$8,ideal_bonus_pct=$9,is_active=$10,updated_at=NOW()
RETURNING id,group_goal_id,customer_code,representative_code,minimum_value,minimum_bonus_pct,probable_value,probable_bonus_pct,ideal_value,ideal_bonus_pct,is_active,created_at,updated_at`,
		customer.GroupGoalID, customer.CustomerCode, customer.RepresentativeCode, customer.MinimumValue, customer.MinimumBonusPct, customer.ProbableValue, customer.ProbableBonusPct, customer.IdealValue, customer.IdealBonusPct, customer.IsActive)
	return scanGroupCustomer(row)
}

func (r *Repository) UpsertBalance(ctx context.Context, balance *entity.Balance) (*entity.Balance, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO public.sales_goal_balances (period_code,next_period_code,balance_scope,representative_code,commercial_group_code,customer_code,goal_type,realized_value,ideal_value,balance_value,notes)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
RETURNING id,period_code,next_period_code,balance_scope,representative_code,commercial_group_code,customer_code,goal_type,realized_value,ideal_value,balance_value,notes,created_at,updated_at`,
		balance.PeriodCode, balance.NextPeriodCode, balance.BalanceScope, balance.RepresentativeCode, balance.CommercialGroupCode, balance.CustomerCode, balance.GoalType, balance.RealizedValue, balance.IdealValue, balance.BalanceValue, balance.Notes)
	return scanBalance(row)
}

func (r *Repository) Report(ctx context.Context, filter goalrepo.ReportFilter) ([]goalrepo.ReportRow, error) {
	sqlText, args := reportSQL(filter)
	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []goalrepo.ReportRow{}
	for rows.Next() {
		var row goalrepo.ReportRow
		var rep, group, customer sql.NullInt64
		err := rows.Scan(&row.Scope, &rep, &group, &customer, &row.PeriodCode, &row.PeriodDescription, &row.AnalysisBase, &row.TargetValue, &row.TargetQuantity, &row.RealizedValue, &row.RealizedQuantity, &row.BalanceValue, &row.AchievementPct, &row.BonusPct, &row.Status)
		if err != nil {
			return nil, err
		}
		row.RepresentativeCode = intPtr(rep)
		row.CommercialGroupCode = intPtr(group)
		row.CustomerCode = intPtr(customer)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) listItems(ctx context.Context, goalCode int64) ([]*entity.GoalItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,goal_code,target_type,item_code,item_classification_code,item_group_code,sales_uom,target_quantity,target_value,bonus_pct,is_active,created_at,updated_at FROM public.sales_goal_items WHERE goal_code=$1 ORDER BY id`, goalCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.GoalItem{}
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func reportSQL(filter goalrepo.ReportFilter) (string, []any) {
	args := []any{}
	add := func(where *string, clause string, arg any) {
		args = append(args, arg)
		*where += fmt.Sprintf(" AND "+clause, len(args))
	}
	goalWhere := "WHERE g.is_active=TRUE"
	groupWhere := "WHERE gt.is_active=TRUE"
	orderWhere := "WHERE so.is_active=TRUE AND so.status <> 'C'"
	if filter.RepresentativeCode != nil {
		add(&goalWhere, "g.representative_code=$%d", *filter.RepresentativeCode)
		add(&orderWhere, "so.representative_code=$%d", *filter.RepresentativeCode)
	}
	if filter.CustomerCode != nil {
		add(&orderWhere, "so.customer_code=$%d", *filter.CustomerCode)
		add(&groupWhere, "EXISTS (SELECT 1 FROM public.sales_goal_group_customers gc_filter WHERE gc_filter.group_goal_id=gt.id AND gc_filter.customer_code=$%d)", *filter.CustomerCode)
	}
	if filter.RegionCode != nil {
		add(&goalWhere, "EXISTS (SELECT 1 FROM public.representative_regions rr_filter WHERE rr_filter.representative_code=g.representative_code AND rr_filter.is_active=TRUE AND rr_filter.region_code=$%d)", *filter.RegionCode)
		add(&groupWhere, "EXISTS (SELECT 1 FROM public.sales_goal_group_customers gc_filter JOIN public.representative_regions rr_filter ON rr_filter.representative_code=gc_filter.representative_code WHERE gc_filter.group_goal_id=gt.id AND rr_filter.is_active=TRUE AND rr_filter.region_code=$%d)", *filter.RegionCode)
	}
	if filter.MicroregionCode != nil {
		add(&goalWhere, "EXISTS (SELECT 1 FROM public.representative_regions rr_filter WHERE rr_filter.representative_code=g.representative_code AND rr_filter.is_active=TRUE AND rr_filter.microregion_code=$%d)", *filter.MicroregionCode)
		add(&groupWhere, "EXISTS (SELECT 1 FROM public.sales_goal_group_customers gc_filter JOIN public.representative_regions rr_filter ON rr_filter.representative_code=gc_filter.representative_code WHERE gc_filter.group_goal_id=gt.id AND rr_filter.is_active=TRUE AND rr_filter.microregion_code=$%d)", *filter.MicroregionCode)
	}
	if filter.PeriodCode != nil {
		add(&goalWhere, "g.period_code=$%d", *filter.PeriodCode)
		add(&groupWhere, "gt.period_code=$%d", *filter.PeriodCode)
	}
	if filter.AnalysisBase != "" {
		add(&goalWhere, "g.analysis_base=$%d", filter.AnalysisBase)
		add(&groupWhere, "gt.goal_type=$%d", filter.AnalysisBase)
	}
	if filter.From != nil {
		add(&goalWhere, "p.end_date >= $%d", *filter.From)
		add(&groupWhere, "pg.end_date >= $%d", *filter.From)
		add(&orderWhere, "COALESCE(so.sale_date, so.emission_date) >= $%d", *filter.From)
	}
	if filter.To != nil {
		add(&goalWhere, "p.start_date <= $%d", *filter.To)
		add(&groupWhere, "pg.start_date <= $%d", *filter.To)
		add(&orderWhere, "COALESCE(so.sale_date, so.emission_date) <= $%d", *filter.To)
	}
	orderCTE := `orders AS (
SELECT so.representative_code, so.customer_code, COALESCE(so.sale_date, so.emission_date) AS ref_date,
       SUM(COALESCE(so.total_net,0)) AS realized_value,
       SUM(COALESCE(soi.requested_qty,0)) AS realized_quantity
FROM public.sales_orders so
LEFT JOIN public.sales_order_items soi ON soi.sales_order_code=so.code AND soi.is_active=TRUE
` + orderWhere + `
GROUP BY so.representative_code, so.customer_code, COALESCE(so.sale_date, so.emission_date)
)`
	sqlText := `WITH ` + orderCTE + `,
rep_goals AS (
SELECT 'REPRESENTATIVE'::text AS scope, g.representative_code, NULL::bigint AS commercial_group_code, NULL::bigint AS customer_code,
       p.code AS period_code, p.description AS period_description, g.analysis_base,
       COALESCE(SUM(gi.target_value),0) AS target_value,
       COALESCE(SUM(gi.target_quantity),0) AS target_quantity,
       COALESCE((SELECT SUM(o.realized_value) FROM orders o WHERE o.representative_code=g.representative_code AND o.ref_date BETWEEN p.start_date AND p.end_date),0) AS realized_value,
       COALESCE((SELECT SUM(o.realized_quantity) FROM orders o WHERE o.representative_code=g.representative_code AND o.ref_date BETWEEN p.start_date AND p.end_date),0) AS realized_quantity,
       COALESCE((SELECT SUM(b.balance_value) FROM public.sales_goal_balances b WHERE b.period_code=p.code AND b.representative_code=g.representative_code AND b.goal_type=g.analysis_base),0) AS balance_value,
       GREATEST(g.award_pct, COALESCE(MAX(gi.bonus_pct),0)) AS bonus_pct
FROM public.sales_goals g
JOIN public.sales_goal_periods p ON p.code=g.period_code
LEFT JOIN public.sales_goal_items gi ON gi.goal_code=g.code AND gi.is_active=TRUE
` + goalWhere + `
GROUP BY g.representative_code,p.code,p.description,p.start_date,p.end_date,g.analysis_base,g.award_pct
),
group_goals AS (
SELECT 'GROUP'::text AS scope, NULL::bigint AS representative_code, gt.commercial_group_code, NULL::bigint AS customer_code,
       pg.code AS period_code, pg.description AS period_description, gt.goal_type AS analysis_base,
       gt.ideal_value AS target_value, 0::numeric AS target_quantity,
       COALESCE(SUM(o.realized_value),0) AS realized_value, COALESCE(SUM(o.realized_quantity),0) AS realized_quantity,
       COALESCE((SELECT SUM(b.balance_value) FROM public.sales_goal_balances b WHERE b.period_code=pg.code AND b.commercial_group_code=gt.commercial_group_code AND b.goal_type=gt.goal_type),0) AS balance_value,
       CASE WHEN COALESCE(SUM(o.realized_value),0) >= gt.ideal_value THEN gt.ideal_bonus_pct WHEN COALESCE(SUM(o.realized_value),0) >= gt.probable_value THEN gt.probable_bonus_pct WHEN COALESCE(SUM(o.realized_value),0) >= gt.minimum_value THEN gt.minimum_bonus_pct ELSE 0 END AS bonus_pct
FROM public.sales_goal_group_targets gt
JOIN public.sales_goal_periods pg ON pg.code=gt.period_code
LEFT JOIN public.sales_goal_group_customers gc ON gc.group_goal_id=gt.id AND gc.is_active=TRUE
LEFT JOIN orders o ON o.customer_code=gc.customer_code AND o.ref_date BETWEEN pg.start_date AND pg.end_date
` + groupWhere + `
GROUP BY gt.commercial_group_code,pg.code,pg.description,gt.goal_type,gt.minimum_value,gt.minimum_bonus_pct,gt.probable_value,gt.probable_bonus_pct,gt.ideal_value,gt.ideal_bonus_pct
)
SELECT scope, representative_code, commercial_group_code, customer_code, period_code, period_description, analysis_base,
       target_value, target_quantity, realized_value, realized_quantity, balance_value,
       CASE WHEN target_value > 0 THEN ROUND(((realized_value + balance_value) / target_value) * 100, 4) ELSE 0 END AS achievement_pct,
       bonus_pct,
       CASE WHEN target_value = 0 THEN 'NO_TARGET' WHEN realized_value + balance_value >= target_value THEN 'ACHIEVED' ELSE 'OPEN' END AS status
FROM rep_goals
UNION ALL
SELECT scope, representative_code, commercial_group_code, customer_code, period_code, period_description, analysis_base,
       target_value, target_quantity, realized_value, realized_quantity, balance_value,
       CASE WHEN target_value > 0 THEN ROUND(((realized_value + balance_value) / target_value) * 100, 4) ELSE 0 END AS achievement_pct,
       bonus_pct,
       CASE WHEN target_value = 0 THEN 'NO_TARGET' WHEN realized_value + balance_value >= target_value THEN 'ACHIEVED' ELSE 'OPEN' END AS status
FROM group_goals
ORDER BY period_code, scope, representative_code NULLS LAST, commercial_group_code NULLS LAST`
	return sqlText, args
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPeriod(row rowScanner) (*entity.Period, error) {
	var p entity.Period
	err := row.Scan(&p.Code, &p.Description, &p.PeriodType, &p.StartDate, &p.EndDate, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	return &p, normalizeErr(err)
}

func scanGoal(row rowScanner) (*entity.Goal, error) {
	var g entity.Goal
	var notes sql.NullString
	err := row.Scan(&g.Code, &g.RepresentativeCode, &g.PeriodCode, &g.AnalysisBase, &g.AwardPct, &notes, &g.IsActive, &g.CreatedAt, &g.UpdatedAt)
	g.Notes = strPtr(notes)
	return &g, normalizeErr(err)
}

func scanItem(row rowScanner) (*entity.GoalItem, error) {
	var i entity.GoalItem
	var item, class, group sql.NullInt64
	var uom sql.NullString
	err := row.Scan(&i.ID, &i.GoalCode, &i.TargetType, &item, &class, &group, &uom, &i.TargetQuantity, &i.TargetValue, &i.BonusPct, &i.IsActive, &i.CreatedAt, &i.UpdatedAt)
	i.ItemCode = intPtr(item)
	i.ItemClassificationCode = intPtr(class)
	i.ItemGroupCode = intPtr(group)
	i.SalesUOM = strPtr(uom)
	return &i, normalizeErr(err)
}

func scanGroupTarget(row rowScanner) (*entity.GroupTarget, error) {
	var g entity.GroupTarget
	err := row.Scan(&g.ID, &g.PeriodCode, &g.CommercialGroupCode, &g.GoalType, &g.MinimumValue, &g.MinimumBonusPct, &g.ProbableValue, &g.ProbableBonusPct, &g.IdealValue, &g.IdealBonusPct, &g.IsActive, &g.CreatedAt, &g.UpdatedAt)
	return &g, normalizeErr(err)
}

func scanGroupCustomer(row rowScanner) (*entity.GroupCustomer, error) {
	var c entity.GroupCustomer
	var rep sql.NullInt64
	err := row.Scan(&c.ID, &c.GroupGoalID, &c.CustomerCode, &rep, &c.MinimumValue, &c.MinimumBonusPct, &c.ProbableValue, &c.ProbableBonusPct, &c.IdealValue, &c.IdealBonusPct, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	c.RepresentativeCode = intPtr(rep)
	return &c, normalizeErr(err)
}

func scanBalance(row rowScanner) (*entity.Balance, error) {
	var b entity.Balance
	var next, rep, group, customer sql.NullInt64
	var notes sql.NullString
	err := row.Scan(&b.ID, &b.PeriodCode, &next, &b.BalanceScope, &rep, &group, &customer, &b.GoalType, &b.RealizedValue, &b.IdealValue, &b.BalanceValue, &notes, &b.CreatedAt, &b.UpdatedAt)
	b.NextPeriodCode = intPtr(next)
	b.RepresentativeCode = intPtr(rep)
	b.CommercialGroupCode = intPtr(group)
	b.CustomerCode = intPtr(customer)
	b.Notes = strPtr(notes)
	return &b, normalizeErr(err)
}

func normalizeErr(err error) error {
	if err == nil {
		return nil
	}
	if err == pgx.ErrNoRows {
		return fmt.Errorf("sales goal record not found")
	}
	return err
}

func intPtr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func strPtr(v sql.NullString) *string {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return nil
	}
	return &v.String
}
