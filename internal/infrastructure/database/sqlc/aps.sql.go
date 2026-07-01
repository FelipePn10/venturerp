package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// ─── production_sequences ─────────────────────────────────────────────────────

const insertProductionSequence = `INSERT INTO production_sequences
(production_order_id, operation_id, work_center_id, sequence_position, scheduled_start, scheduled_end, status)
VALUES ($1,$2,$3,$4,$5,$6,$7)
RETURNING id, production_order_id, operation_id, work_center_id, sequence_position,
          scheduled_start, scheduled_end, status, created_at, updated_at`

type InsertProductionSequenceParams struct {
	ProductionOrderID int64
	OperationID       pgtype.Int8
	WorkCenterID      int64
	SequencePosition  int32
	ScheduledStart    pgtype.Timestamptz
	ScheduledEnd      pgtype.Timestamptz
	Status            string
}

type DBProductionSequence struct {
	ID                int64
	ProductionOrderID int64
	OperationID       pgtype.Int8
	WorkCenterID      int64
	SequencePosition  int32
	ScheduledStart    pgtype.Timestamptz
	ScheduledEnd      pgtype.Timestamptz
	Status            string
	CreatedAt         pgtype.Timestamptz
	UpdatedAt         pgtype.Timestamptz
}

func (q *Queries) InsertProductionSequence(ctx context.Context, arg InsertProductionSequenceParams) (DBProductionSequence, error) {
	row := q.db.QueryRow(ctx, insertProductionSequence,
		arg.ProductionOrderID, arg.OperationID, arg.WorkCenterID, arg.SequencePosition,
		arg.ScheduledStart, arg.ScheduledEnd, arg.Status)
	var i DBProductionSequence
	err := row.Scan(&i.ID, &i.ProductionOrderID, &i.OperationID, &i.WorkCenterID,
		&i.SequencePosition, &i.ScheduledStart, &i.ScheduledEnd, &i.Status, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

const listSequencesByOrder = `SELECT id, production_order_id, operation_id, work_center_id, sequence_position,
scheduled_start, scheduled_end, status, created_at, updated_at
FROM production_sequences WHERE production_order_id=$1 ORDER BY sequence_position`

func (q *Queries) ListSequencesByOrder(ctx context.Context, orderID int64) ([]DBProductionSequence, error) {
	rows, err := q.db.Query(ctx, listSequencesByOrder, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSequences(rows)
}

const listSequencesByWorkCenter = `SELECT id, production_order_id, operation_id, work_center_id, sequence_position,
scheduled_start, scheduled_end, status, created_at, updated_at
FROM production_sequences WHERE work_center_id=$1 AND scheduled_start >= $2 AND scheduled_end <= $3
ORDER BY work_center_id, scheduled_start`

func (q *Queries) ListSequencesByWorkCenter(ctx context.Context, workCenterID int64, from, to pgtype.Timestamptz) ([]DBProductionSequence, error) {
	rows, err := q.db.Query(ctx, listSequencesByWorkCenter, workCenterID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSequences(rows)
}

const getProductionSequence = `SELECT id, production_order_id, operation_id, work_center_id, sequence_position,
scheduled_start, scheduled_end, status, created_at, updated_at
FROM production_sequences WHERE id=$1`

func (q *Queries) GetProductionSequence(ctx context.Context, id int64) (DBProductionSequence, error) {
	row := q.db.QueryRow(ctx, getProductionSequence, id)
	var i DBProductionSequence
	err := row.Scan(&i.ID, &i.ProductionOrderID, &i.OperationID, &i.WorkCenterID,
		&i.SequencePosition, &i.ScheduledStart, &i.ScheduledEnd, &i.Status, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

const updateProductionSequence = `UPDATE production_sequences
SET work_center_id=$2, scheduled_start=$3, scheduled_end=$4, updated_at=NOW()
WHERE id=$1
RETURNING id, production_order_id, operation_id, work_center_id, sequence_position,
          scheduled_start, scheduled_end, status, created_at, updated_at`

type UpdateProductionSequenceParams struct {
	ID             int64
	WorkCenterID   int64
	ScheduledStart pgtype.Timestamptz
	ScheduledEnd   pgtype.Timestamptz
}

func (q *Queries) UpdateProductionSequence(ctx context.Context, arg UpdateProductionSequenceParams) (DBProductionSequence, error) {
	row := q.db.QueryRow(ctx, updateProductionSequence,
		arg.ID, arg.WorkCenterID, arg.ScheduledStart, arg.ScheduledEnd)
	var i DBProductionSequence
	err := row.Scan(&i.ID, &i.ProductionOrderID, &i.OperationID, &i.WorkCenterID,
		&i.SequencePosition, &i.ScheduledStart, &i.ScheduledEnd, &i.Status, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

const deleteSequencesByOrder = `DELETE FROM production_sequences WHERE production_order_id=$1`

func (q *Queries) DeleteSequencesByOrder(ctx context.Context, orderID int64) error {
	_, err := q.db.Exec(ctx, deleteSequencesByOrder, orderID)
	return err
}

func scanSequences(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]DBProductionSequence, error) {
	var items []DBProductionSequence
	for rows.Next() {
		var i DBProductionSequence
		if err := rows.Scan(&i.ID, &i.ProductionOrderID, &i.OperationID, &i.WorkCenterID,
			&i.SequencePosition, &i.ScheduledStart, &i.ScheduledEnd, &i.Status, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ─── helpers for APS calculation ──────────────────────────────────────────────

// priority is a free-text VARCHAR (e.g. "NORMAL", "ALTA", or a numeric string),
// so a blind ::int cast raised "invalid input syntax for integer: NORMAL"
// (SQLSTATE 22P02). Map it to a numeric rank: numeric strings keep their value,
// known textual buckets get a rank, and anything else defaults to the middle.
const getOpenProductionOrders = `
SELECT id,
       CASE
           WHEN priority ~ '^[0-9]+$' THEN priority::int
           WHEN upper(priority) IN ('ALTA', 'HIGH', 'URGENTE', 'URGENT') THEN 1
           WHEN upper(priority) IN ('BAIXA', 'LOW') THEN 9
           ELSE 5
       END AS priority,
       COALESCE(start_date, end_date)::timestamptz AS planned_date
FROM production_orders
WHERE status IN ('OPEN', 'IN_PROGRESS') AND is_active = TRUE
ORDER BY 2 ASC, planned_date ASC`

type DBOpenProductionOrder struct {
	ID          int64
	Priority    int32
	PlannedDate pgtype.Timestamptz
}

func (q *Queries) GetOpenProductionOrders(ctx context.Context) ([]DBOpenProductionOrder, error) {
	rows, err := q.db.Query(ctx, getOpenProductionOrders)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBOpenProductionOrder
	for rows.Next() {
		var i DBOpenProductionOrder
		if err := rows.Scan(&i.ID, &i.Priority, &i.PlannedDate); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const getOrderOperations = `
SELECT id, sequence, work_center_id, planned_hours, setup_hours
FROM production_order_operations
WHERE production_order_id=$1 AND status NOT IN ('DONE','SKIPPED')
ORDER BY sequence`

type DBOrderOperation struct {
	ID           int64
	Sequence     int32
	WorkCenterID pgtype.Int8
	PlannedHours pgtype.Numeric
	SetupHours   pgtype.Numeric
}

func (q *Queries) GetOrderOperations(ctx context.Context, orderID int64) ([]DBOrderOperation, error) {
	rows, err := q.db.Query(ctx, getOrderOperations, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBOrderOperation
	for rows.Next() {
		var i DBOrderOperation
		if err := rows.Scan(&i.ID, &i.Sequence, &i.WorkCenterID, &i.PlannedHours, &i.SetupHours); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
