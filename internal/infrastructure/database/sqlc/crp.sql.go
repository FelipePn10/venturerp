package sqlc

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ─── capacity_requirements ────────────────────────────────────────────────────

const upsertCapacityRequirement = `INSERT INTO capacity_requirements
(plan_code, work_center_id, req_date, required_hours, available_hours)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (plan_code, work_center_id, req_date) DO UPDATE SET
    required_hours  = EXCLUDED.required_hours,
    available_hours = EXCLUDED.available_hours
RETURNING id, plan_code, work_center_id, req_date, required_hours, available_hours, load_pct, created_at`

type UpsertCapacityRequirementParams struct {
	PlanCode       int64
	WorkCenterID   int64
	ReqDate        pgtype.Date
	RequiredHours  pgtype.Numeric
	AvailableHours pgtype.Numeric
}

type DBCapacityRequirement struct {
	ID             int64
	PlanCode       int64
	WorkCenterID   int64
	ReqDate        pgtype.Date
	RequiredHours  pgtype.Numeric
	AvailableHours pgtype.Numeric
	LoadPct        pgtype.Numeric
	CreatedAt      pgtype.Timestamptz
}

func (q *Queries) UpsertCapacityRequirement(ctx context.Context, arg UpsertCapacityRequirementParams) (DBCapacityRequirement, error) {
	row := q.db.QueryRow(ctx, upsertCapacityRequirement,
		arg.PlanCode, arg.WorkCenterID, arg.ReqDate, arg.RequiredHours, arg.AvailableHours)
	var i DBCapacityRequirement
	err := row.Scan(&i.ID, &i.PlanCode, &i.WorkCenterID, &i.ReqDate,
		&i.RequiredHours, &i.AvailableHours, &i.LoadPct, &i.CreatedAt)
	return i, err
}

const listCRPByPlan = `SELECT id, plan_code, work_center_id, req_date, required_hours, available_hours, load_pct, created_at
FROM capacity_requirements WHERE plan_code=$1 ORDER BY req_date, work_center_id`

func (q *Queries) ListCRPByPlan(ctx context.Context, planCode int64) ([]DBCapacityRequirement, error) {
	rows, err := q.db.Query(ctx, listCRPByPlan, planCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCRPRows(rows)
}

const listOverloadedCRPByPlan = `SELECT id, plan_code, work_center_id, req_date, required_hours, available_hours, load_pct, created_at
FROM capacity_requirements WHERE plan_code=$1 AND load_pct > 100 ORDER BY load_pct DESC`

func (q *Queries) ListOverloadedCRPByPlan(ctx context.Context, planCode int64) ([]DBCapacityRequirement, error) {
	rows, err := q.db.Query(ctx, listOverloadedCRPByPlan, planCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCRPRows(rows)
}

const listCRPByWorkCenter = `SELECT id, plan_code, work_center_id, req_date, required_hours, available_hours, load_pct, created_at
FROM capacity_requirements WHERE work_center_id=$1 AND req_date BETWEEN $2 AND $3 ORDER BY req_date`

func (q *Queries) ListCRPByWorkCenter(ctx context.Context, workCenterID int64, from, to pgtype.Date) ([]DBCapacityRequirement, error) {
	rows, err := q.db.Query(ctx, listCRPByWorkCenter, workCenterID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCRPRows(rows)
}

const deleteCRPByPlan = `DELETE FROM capacity_requirements WHERE plan_code=$1`

func (q *Queries) DeleteCRPByPlan(ctx context.Context, planCode int64) error {
	_, err := q.db.Exec(ctx, deleteCRPByPlan, planCode)
	return err
}

func scanCRPRows(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]DBCapacityRequirement, error) {
	var items []DBCapacityRequirement
	for rows.Next() {
		var i DBCapacityRequirement
		if err := rows.Scan(&i.ID, &i.PlanCode, &i.WorkCenterID, &i.ReqDate,
			&i.RequiredHours, &i.AvailableHours, &i.LoadPct, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ─── helpers for CRP calculation ──────────────────────────────────────────────

// GetPlannedOrdersForCRP returns planned orders (with optional standard route_id) for a given MRP plan.
const getPlannedOrdersForCRP = `
SELECT po.id, po.item_code,
       COALESCE(po.quantity_corrected, po.quantity, 0)::double precision AS quantity,
       COALESCE(po.start_date, po.need_date) AS planned_date,
       mr.id AS route_id
FROM planned_orders po
LEFT JOIN manufacturing_routes mr ON mr.item_code = po.item_code
    AND mr.is_standard = TRUE AND mr.is_active = TRUE
WHERE po.plan_code = $1
  AND po.is_active = TRUE
  AND po.status NOT IN ('CANCELLED')
ORDER BY planned_date`

type DBPlannedOrderForCRP struct {
	ID          int64
	ItemCode    int64
	Quantity    float64
	PlannedDate pgtype.Date
	RouteID     pgtype.Int8
}

func (q *Queries) GetPlannedOrdersForCRP(ctx context.Context, planCode int64) ([]DBPlannedOrderForCRP, error) {
	rows, err := q.db.Query(ctx, getPlannedOrdersForCRP, planCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBPlannedOrderForCRP
	for rows.Next() {
		var i DBPlannedOrderForCRP
		if err := rows.Scan(&i.ID, &i.ItemCode, &i.Quantity, &i.PlannedDate, &i.RouteID); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// GetRouteOpHoursForCRP returns (work_center_id, effective_hours) per route operation.
const getRouteOpHoursForCRP = `
SELECT ro.work_center_id,
       COALESCE(ro.setup_time, o.setup_time, 0) + COALESCE(ro.standard_time, o.standard_time, 0) AS eff_hours
FROM route_operations ro
JOIN operations o ON o.id = ro.operation_id
WHERE ro.route_id = $1 AND ro.is_active = TRUE`

type DBRouteOpHours struct {
	WorkCenterID pgtype.Int8
	EffHours     float64
}

func (q *Queries) GetRouteOpHoursForCRP(ctx context.Context, routeID int64) ([]DBRouteOpHours, error) {
	rows, err := q.db.Query(ctx, getRouteOpHoursForCRP, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBRouteOpHours
	for rows.Next() {
		var i DBRouteOpHours
		if err := rows.Scan(&i.WorkCenterID, &i.EffHours); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// GetMachineAvailableHours returns available hours per day for a work center.
// Uses machine count × 8h as a conservative baseline; override via machine schedules.
const getMachineAvailableHours = `
SELECT COALESCE(COUNT(*) * 8.0, 8.0)
FROM machines m
WHERE m.machine_type_code = $1 AND m.is_active = TRUE`

func (q *Queries) GetMachineAvailableHours(ctx context.Context, workCenterID int64) (float64, error) {
	row := q.db.QueryRow(ctx, getMachineAvailableHours, workCenterID)
	var h float64
	return h, row.Scan(&h)
}

// ─── pgtype date helper ───────────────────────────────────────────────────────

func ToPgDate(t time.Time) pgtype.Date {
	return pgtype.Date{
		Time:  t,
		Valid: true,
	}
}

func FromPgDate(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}
