package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// Queries backing the monthly production-schedule board (Gantt). They are
// hand-written (like aps.sql.go) so they can join across production_sequences,
// production_orders, machine_types and production_order_operations without a
// codegen pass.

// ─── scheduled bars (APS sequences) ───────────────────────────────────────────

// A bar is selected when it overlaps the half-open window [from, to):
//
//	scheduled_start < to  AND  scheduled_end >= from
const listGanttScheduledBars = `
SELECT ps.id,
       ps.production_order_id,
       po.order_number,
       po.item_code,
       po.mask,
       ps.work_center_id,
       COALESCE(mt.name, '')            AS work_center_name,
       ps.operation_id,
       COALESCE(poo.operation_name, '') AS operation_name,
       ps.sequence_position,
       ps.scheduled_start,
       ps.scheduled_end,
       ps.status,
       COALESCE(po.priority, '')        AS priority,
       po.planned_qty,
       po.produced_qty,
       po.status                        AS order_status,
       po.end_date,
       COALESCE(poo.status, '')         AS op_status,
       COALESCE(poo.actual_hours, 0)    AS op_actual_hours,
       COALESCE(poo.planned_hours, 0)   AS op_planned_hours
FROM production_sequences ps
JOIN production_orders po ON po.id = ps.production_order_id
LEFT JOIN machine_types mt ON mt.id = ps.work_center_id
LEFT JOIN production_order_operations poo ON poo.id = ps.operation_id
WHERE ps.scheduled_start < $2 AND ps.scheduled_end >= $1
ORDER BY ps.work_center_id, ps.scheduled_start`

type DBGanttScheduledBar struct {
	ID                int64
	ProductionOrderID int64
	OrderNumber       int64
	ItemCode          int64
	Mask              string
	WorkCenterID      int64
	WorkCenterName    string
	OperationID       pgtype.Int8
	OperationName     string
	SequencePosition  int32
	ScheduledStart    pgtype.Timestamptz
	ScheduledEnd      pgtype.Timestamptz
	Status            string
	Priority          string
	PlannedQty        pgtype.Numeric
	ProducedQty       pgtype.Numeric
	OrderStatus       string
	EndDate           pgtype.Date
	OpStatus          string
	OpActualHours     pgtype.Numeric
	OpPlannedHours    pgtype.Numeric
}

func (q *Queries) ListGanttScheduledBars(ctx context.Context, from, to pgtype.Timestamptz) ([]DBGanttScheduledBar, error) {
	rows, err := q.db.Query(ctx, listGanttScheduledBars, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBGanttScheduledBar
	for rows.Next() {
		var i DBGanttScheduledBar
		if err := rows.Scan(
			&i.ID, &i.ProductionOrderID, &i.OrderNumber, &i.ItemCode, &i.Mask,
			&i.WorkCenterID, &i.WorkCenterName, &i.OperationID, &i.OperationName,
			&i.SequencePosition, &i.ScheduledStart, &i.ScheduledEnd, &i.Status,
			&i.Priority, &i.PlannedQty, &i.ProducedQty, &i.OrderStatus, &i.EndDate,
			&i.OpStatus, &i.OpActualHours, &i.OpPlannedHours,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ─── fallback bars (orders not yet APS-sequenced) ─────────────────────────────

// Orders that have no production_sequences are plotted from their own dates, so
// the board never looks empty before sequencing has run. The bar spans
// [start_date, end_date] (whichever is present), clamped by the caller.
const listGanttFallbackBars = `
SELECT po.id,
       po.order_number,
       po.item_code,
       po.mask,
       COALESCE(po.machine_id, 0) AS machine_id,
       po.start_date,
       po.end_date,
       po.status,
       COALESCE(po.priority, '')  AS priority,
       po.planned_qty,
       po.produced_qty
FROM production_orders po
WHERE po.is_active = TRUE
  AND po.status IN ('OPEN', 'IN_PROGRESS', 'COMPLETED')
  AND NOT EXISTS (SELECT 1 FROM production_sequences ps WHERE ps.production_order_id = po.id)
  AND COALESCE(po.start_date, po.end_date) IS NOT NULL
  AND COALESCE(po.start_date, po.end_date) < $2::date
  AND COALESCE(po.end_date, po.start_date) >= $1::date
ORDER BY COALESCE(po.start_date, po.end_date), po.order_number`

type DBGanttFallbackBar struct {
	ID          int64
	OrderNumber int64
	ItemCode    int64
	Mask        string
	MachineID   int64
	StartDate   pgtype.Date
	EndDate     pgtype.Date
	Status      string
	Priority    string
	PlannedQty  pgtype.Numeric
	ProducedQty pgtype.Numeric
}

func (q *Queries) ListGanttFallbackBars(ctx context.Context, from, to pgtype.Date) ([]DBGanttFallbackBar, error) {
	rows, err := q.db.Query(ctx, listGanttFallbackBars, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBGanttFallbackBar
	for rows.Next() {
		var i DBGanttFallbackBar
		if err := rows.Scan(
			&i.ID, &i.OrderNumber, &i.ItemCode, &i.Mask, &i.MachineID,
			&i.StartDate, &i.EndDate, &i.Status, &i.Priority, &i.PlannedQty, &i.ProducedQty,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ─── finish-start dependencies ────────────────────────────────────────────────

// Explicit predecessor→successor edges from route_operation_network, mapped onto
// the scheduled bars: route operations → the order's mirrored operations
// (production_order_operations.route_operation_id) → their production_sequences.
// Both endpoints are constrained to the same production order and to the window.
const listGanttDependencies = `
SELECT ps_pred.id          AS from_seq,
       ps_succ.id          AS to_seq,
       COALESCE(ron.overlap_pct, 0) AS overlap_pct
FROM route_operation_network ron
JOIN production_order_operations poo_p ON poo_p.route_operation_id = ron.predecessor_id
JOIN production_order_operations poo_s ON poo_s.route_operation_id = ron.successor_id
                                       AND poo_s.production_order_id = poo_p.production_order_id
JOIN production_sequences ps_pred ON ps_pred.operation_id = poo_p.id
JOIN production_sequences ps_succ ON ps_succ.operation_id = poo_s.id
WHERE ps_pred.scheduled_start < $2 AND ps_pred.scheduled_end >= $1
  AND ps_succ.scheduled_start < $2 AND ps_succ.scheduled_end >= $1`

// listGanttOrderDependencies is the same edge mapping scoped to a single order,
// used by the cascade reschedule (no window filter so the whole chain is visible).
const listGanttOrderDependencies = `
SELECT ps_pred.id          AS from_seq,
       ps_succ.id          AS to_seq,
       COALESCE(ron.overlap_pct, 0) AS overlap_pct
FROM route_operation_network ron
JOIN production_order_operations poo_p ON poo_p.route_operation_id = ron.predecessor_id
JOIN production_order_operations poo_s ON poo_s.route_operation_id = ron.successor_id
                                       AND poo_s.production_order_id = poo_p.production_order_id
JOIN production_sequences ps_pred ON ps_pred.operation_id = poo_p.id
JOIN production_sequences ps_succ ON ps_succ.operation_id = poo_s.id
WHERE poo_p.production_order_id = $1`

type DBGanttDependency struct {
	FromSeq    int64
	ToSeq      int64
	OverlapPct pgtype.Numeric
}

func scanGanttDependencies(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]DBGanttDependency, error) {
	var items []DBGanttDependency
	for rows.Next() {
		var i DBGanttDependency
		if err := rows.Scan(&i.FromSeq, &i.ToSeq, &i.OverlapPct); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (q *Queries) ListGanttDependencies(ctx context.Context, from, to pgtype.Timestamptz) ([]DBGanttDependency, error) {
	rows, err := q.db.Query(ctx, listGanttDependencies, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGanttDependencies(rows)
}

func (q *Queries) ListGanttOrderDependencies(ctx context.Context, orderID int64) ([]DBGanttDependency, error) {
	rows, err := q.db.Query(ctx, listGanttOrderDependencies, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGanttDependencies(rows)
}

// ─── per-resource capacity load (CRP) ─────────────────────────────────────────

// Capacity load is aggregated across every MRP plan: required hours sum up, the
// available hours take the per-work-center maximum (the same capacity is mirrored
// per plan, so MAX avoids double counting).
const listGanttResourceLoad = `
SELECT cr.work_center_id,
       cr.req_date,
       SUM(cr.required_hours)  AS required_hours,
       MAX(cr.available_hours) AS available_hours
FROM capacity_requirements cr
WHERE cr.req_date >= $1::date AND cr.req_date < $2::date
GROUP BY cr.work_center_id, cr.req_date
ORDER BY cr.work_center_id, cr.req_date`

type DBGanttResourceLoad struct {
	WorkCenterID   int64
	ReqDate        pgtype.Date
	RequiredHours  pgtype.Numeric
	AvailableHours pgtype.Numeric
}

func (q *Queries) ListGanttResourceLoad(ctx context.Context, from, to pgtype.Date) ([]DBGanttResourceLoad, error) {
	rows, err := q.db.Query(ctx, listGanttResourceLoad, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBGanttResourceLoad
	for rows.Next() {
		var i DBGanttResourceLoad
		if err := rows.Scan(&i.WorkCenterID, &i.ReqDate, &i.RequiredHours, &i.AvailableHours); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
