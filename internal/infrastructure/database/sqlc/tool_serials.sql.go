package sqlc

// Hand-written (not sqlc-generated) data access for the Ficha de Produção da
// Ferramenta. Mirrors the style of production_order_operations.sql.go so the
// complex read joins (order → operations → tools → serials) stay under our
// control and no sqlc regeneration is required.

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// ─── tool_serials (physical instances) ────────────────────────────────────────

type DBToolSerial struct {
	ID           int64
	ToolID       int64
	SerialNumber string
	Status       string
	LifeUsed     pgtype.Numeric
	Location     pgtype.Text
	Notes        pgtype.Text
	IsActive     bool
	CreatedAt    pgtype.Timestamptz
	UpdatedAt    pgtype.Timestamptz
	CreatedBy    pgtype.UUID
}

const createToolSerial = `INSERT INTO tool_serials
(tool_id, serial_number, status, location, notes, created_by)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING id, tool_id, serial_number, status, life_used, location, notes, is_active, created_at, updated_at, created_by`

type CreateToolSerialParams struct {
	ToolID       int64
	SerialNumber string
	Status       string
	Location     pgtype.Text
	Notes        pgtype.Text
	CreatedBy    pgtype.UUID
}

func (q *Queries) CreateToolSerial(ctx context.Context, arg CreateToolSerialParams) (DBToolSerial, error) {
	row := q.db.QueryRow(ctx, createToolSerial,
		arg.ToolID, arg.SerialNumber, arg.Status, arg.Location, arg.Notes, arg.CreatedBy)
	return scanToolSerial(row)
}

const updateToolSerial = `UPDATE tool_serials SET
	serial_number = $2,
	status        = $3,
	location      = $4,
	notes         = $5,
	updated_at    = NOW()
WHERE id = $1
RETURNING id, tool_id, serial_number, status, life_used, location, notes, is_active, created_at, updated_at, created_by`

type UpdateToolSerialParams struct {
	ID           int64
	SerialNumber string
	Status       string
	Location     pgtype.Text
	Notes        pgtype.Text
}

func (q *Queries) UpdateToolSerial(ctx context.Context, arg UpdateToolSerialParams) (DBToolSerial, error) {
	row := q.db.QueryRow(ctx, updateToolSerial,
		arg.ID, arg.SerialNumber, arg.Status, arg.Location, arg.Notes)
	return scanToolSerial(row)
}

const getToolSerial = `SELECT id, tool_id, serial_number, status, life_used, location, notes, is_active, created_at, updated_at, created_by
FROM tool_serials WHERE id = $1`

func (q *Queries) GetToolSerial(ctx context.Context, id int64) (DBToolSerial, error) {
	row := q.db.QueryRow(ctx, getToolSerial, id)
	return scanToolSerial(row)
}

const listToolSerials = `SELECT id, tool_id, serial_number, status, life_used, location, notes, is_active, created_at, updated_at, created_by
FROM tool_serials
WHERE tool_id = $1 AND ($2::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY serial_number`

type ListToolSerialsParams struct {
	ToolID     int64
	OnlyActive bool
}

func (q *Queries) ListToolSerials(ctx context.Context, arg ListToolSerialsParams) ([]DBToolSerial, error) {
	rows, err := q.db.Query(ctx, listToolSerials, arg.ToolID, arg.OnlyActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBToolSerial
	for rows.Next() {
		ts, err := scanToolSerial(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, ts)
	}
	return items, rows.Err()
}

const deactivateToolSerial = `UPDATE tool_serials SET is_active = FALSE, status = 'INATIVA', updated_at = NOW() WHERE id = $1`

func (q *Queries) DeactivateToolSerial(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, deactivateToolSerial, id)
	return err
}

const consumeToolSerialLife = `UPDATE tool_serials SET life_used = life_used + $2, updated_at = NOW()
WHERE id = $1
RETURNING id, tool_id, serial_number, status, life_used, location, notes, is_active, created_at, updated_at, created_by`

type ConsumeToolSerialLifeParams struct {
	ID       int64
	LifeUsed pgtype.Numeric
}

func (q *Queries) ConsumeToolSerialLife(ctx context.Context, arg ConsumeToolSerialLifeParams) (DBToolSerial, error) {
	row := q.db.QueryRow(ctx, consumeToolSerialLife, arg.ID, arg.LifeUsed)
	return scanToolSerial(row)
}

type toolSerialScanner interface{ Scan(...any) error }

func scanToolSerial(s toolSerialScanner) (DBToolSerial, error) {
	var i DBToolSerial
	err := s.Scan(&i.ID, &i.ToolID, &i.SerialNumber, &i.Status, &i.LifeUsed,
		&i.Location, &i.Notes, &i.IsActive, &i.CreatedAt, &i.UpdatedAt, &i.CreatedBy)
	return i, err
}

// ─── tool production sheet: order header & LOV ────────────────────────────────

// DBToolSheetOrder is the production-order header shown on the sheet and in the
// order list-of-values. OrderType is the planned order's type cast to text
// (NULL for manually-created orders).
type DBToolSheetOrder struct {
	ID          int64
	OrderNumber int64
	OrderType   pgtype.Text
	ItemCode    int64
	ItemName    pgtype.Text
	Mask        string
	PlannedQty  pgtype.Numeric
	Status      string
	StartDate   pgtype.Date
	EndDate     pgtype.Date
}

const getToolSheetOrder = `SELECT po.id, po.order_number, pl.order_type::text, po.item_code,
       i.pdm_description_technique, po.mask, po.planned_qty, po.status, po.start_date, po.end_date
FROM production_orders po
LEFT JOIN planned_orders pl ON pl.id = po.planned_order_id
LEFT JOIN items i ON i.code = po.item_code
WHERE po.id = $1`

func (q *Queries) GetToolSheetOrder(ctx context.Context, orderID int64) (DBToolSheetOrder, error) {
	row := q.db.QueryRow(ctx, getToolSheetOrder, orderID)
	return scanToolSheetOrder(row)
}

// ListEligibleSheetOrders returns active production orders for the LOV, excluding
// OFC-type orders (mapped from the OUTSOURCING planned-order type). When search
// is non-empty it filters by order number (text match) or item code.
const listEligibleSheetOrders = `SELECT po.id, po.order_number, pl.order_type::text, po.item_code,
       i.pdm_description_technique, po.mask, po.planned_qty, po.status, po.start_date, po.end_date
FROM production_orders po
LEFT JOIN planned_orders pl ON pl.id = po.planned_order_id
LEFT JOIN items i ON i.code = po.item_code
WHERE po.is_active = TRUE
  AND (pl.order_type IS NULL OR pl.order_type <> 'OUTSOURCING')
  AND ($1::text = '' OR po.order_number::text ILIKE '%' || $1 || '%'
        OR po.item_code::text ILIKE '%' || $1 || '%'
        OR i.pdm_description_technique ILIKE '%' || $1 || '%')
ORDER BY po.order_number DESC
LIMIT 200`

func (q *Queries) ListEligibleSheetOrders(ctx context.Context, search string) ([]DBToolSheetOrder, error) {
	rows, err := q.db.Query(ctx, listEligibleSheetOrders, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBToolSheetOrder
	for rows.Next() {
		o, err := scanToolSheetOrder(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, o)
	}
	return items, rows.Err()
}

func scanToolSheetOrder(s toolSerialScanner) (DBToolSheetOrder, error) {
	var i DBToolSheetOrder
	err := s.Scan(&i.ID, &i.OrderNumber, &i.OrderType, &i.ItemCode, &i.ItemName,
		&i.Mask, &i.PlannedQty, &i.Status, &i.StartDate, &i.EndDate)
	return i, err
}

// ─── tool production sheet: operations × tools × assigned serial ──────────────

// DBSheetOperationTool is one (operation, tool) row of the sheet. Tool and serial
// columns are nullable: an operation with no route tools yields a single row with
// null tool fields, and an unassigned tool yields null serial fields.
type DBSheetOperationTool struct {
	OperationID     int64
	Sequence        int16
	OperationCode   pgtype.Int8
	OperationName   string
	OperationDesc   pgtype.Text
	WorkCenterID    pgtype.Int8
	ResourceCode    pgtype.Int8
	ResourceName    pgtype.Text
	OperationStatus string

	RouteOpToolID pgtype.Int8
	ToolID        pgtype.Int8
	ToolCode      pgtype.Int8
	ToolName      pgtype.Text
	QtyRequired   pgtype.Numeric

	AssignedSerialID     pgtype.Int8
	AssignedSerialNumber pgtype.Text
	AssignedSerialStatus pgtype.Text
}

const listSheetOperationTools = `SELECT
	poo.id, poo.sequence, o.code, poo.operation_name, o.description,
	poo.work_center_id, mt.code, mt.name, poo.status,
	rot.id, t.id, t.code, t.name, rot.qty_required,
	ts.id, ts.serial_number, ts.status
FROM production_order_operations poo
LEFT JOIN route_operations ro ON ro.id = poo.route_operation_id
LEFT JOIN operations o ON o.id = ro.operation_id
LEFT JOIN machine_types mt ON mt.id = poo.work_center_id
LEFT JOIN route_operation_tools rot ON rot.route_operation_id = ro.id
LEFT JOIN tools t ON t.id = rot.tool_id
LEFT JOIN production_order_operation_tool_serials poots
	ON poots.production_order_operation_id = poo.id AND poots.tool_id = t.id
LEFT JOIN tool_serials ts ON ts.id = poots.tool_serial_id
WHERE poo.production_order_id = $1
ORDER BY poo.sequence, t.code NULLS FIRST`

func (q *Queries) ListSheetOperationTools(ctx context.Context, orderID int64) ([]DBSheetOperationTool, error) {
	rows, err := q.db.Query(ctx, listSheetOperationTools, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBSheetOperationTool
	for rows.Next() {
		var i DBSheetOperationTool
		if err := rows.Scan(
			&i.OperationID, &i.Sequence, &i.OperationCode, &i.OperationName, &i.OperationDesc,
			&i.WorkCenterID, &i.ResourceCode, &i.ResourceName, &i.OperationStatus,
			&i.RouteOpToolID, &i.ToolID, &i.ToolCode, &i.ToolName, &i.QtyRequired,
			&i.AssignedSerialID, &i.AssignedSerialNumber, &i.AssignedSerialStatus,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ─── tool production sheet: assign / substitute ───────────────────────────────

type DBOperationToolSerial struct {
	ID           int64
	OperationID  int64
	ToolID       int64
	ToolSerialID int64
	AssignedAt   pgtype.Timestamptz
	AssignedBy   pgtype.UUID
	UpdatedAt    pgtype.Timestamptz
}

const assignToolSerial = `INSERT INTO production_order_operation_tool_serials
	(production_order_operation_id, tool_id, tool_serial_id, assigned_by)
VALUES ($1,$2,$3,$4)
ON CONFLICT (production_order_operation_id, tool_id)
DO UPDATE SET tool_serial_id = EXCLUDED.tool_serial_id,
              assigned_by    = EXCLUDED.assigned_by,
              updated_at     = NOW()
RETURNING id, production_order_operation_id, tool_id, tool_serial_id, assigned_at, assigned_by, updated_at`

type AssignToolSerialParams struct {
	OperationID  int64
	ToolID       int64
	ToolSerialID int64
	AssignedBy   pgtype.UUID
}

func (q *Queries) AssignToolSerial(ctx context.Context, arg AssignToolSerialParams) (DBOperationToolSerial, error) {
	row := q.db.QueryRow(ctx, assignToolSerial,
		arg.OperationID, arg.ToolID, arg.ToolSerialID, arg.AssignedBy)
	return scanOperationToolSerial(row)
}

const getOperationToolSerial = `SELECT id, production_order_operation_id, tool_id, tool_serial_id, assigned_at, assigned_by, updated_at
FROM production_order_operation_tool_serials
WHERE production_order_operation_id = $1 AND tool_id = $2`

type GetOperationToolSerialParams struct {
	OperationID int64
	ToolID      int64
}

func (q *Queries) GetOperationToolSerial(ctx context.Context, arg GetOperationToolSerialParams) (DBOperationToolSerial, error) {
	row := q.db.QueryRow(ctx, getOperationToolSerial, arg.OperationID, arg.ToolID)
	return scanOperationToolSerial(row)
}

func scanOperationToolSerial(s toolSerialScanner) (DBOperationToolSerial, error) {
	var i DBOperationToolSerial
	err := s.Scan(&i.ID, &i.OperationID, &i.ToolID, &i.ToolSerialID,
		&i.AssignedAt, &i.AssignedBy, &i.UpdatedAt)
	return i, err
}

// ─── tool production sheet: substitution audit trail ──────────────────────────

const recordToolSerialSubstitution = `INSERT INTO tool_serial_substitutions
	(production_order_operation_id, tool_id, old_serial_id, new_serial_id, reason, substituted_by)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING id`

type RecordToolSerialSubstitutionParams struct {
	OperationID   int64
	ToolID        int64
	OldSerialID   pgtype.Int8
	NewSerialID   int64
	Reason        pgtype.Text
	SubstitutedBy pgtype.UUID
}

func (q *Queries) RecordToolSerialSubstitution(ctx context.Context, arg RecordToolSerialSubstitutionParams) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, recordToolSerialSubstitution,
		arg.OperationID, arg.ToolID, arg.OldSerialID, arg.NewSerialID, arg.Reason, arg.SubstitutedBy).Scan(&id)
	return id, err
}

// DBToolSerialSubstitution is a substitution history row with serial numbers and
// the tool denormalized for display.
type DBToolSerialSubstitution struct {
	ID              int64
	OperationID     int64
	ToolID          int64
	ToolCode        int64
	ToolName        string
	OldSerialID     pgtype.Int8
	OldSerialNumber pgtype.Text
	NewSerialID     int64
	NewSerialNumber string
	Reason          pgtype.Text
	SubstitutedAt   pgtype.Timestamptz
	SubstitutedBy   pgtype.UUID
}

const listToolSerialSubstitutions = `SELECT s.id, s.production_order_operation_id, s.tool_id, t.code, t.name,
	s.old_serial_id, os.serial_number, s.new_serial_id, ns.serial_number,
	s.reason, s.substituted_at, s.substituted_by
FROM tool_serial_substitutions s
JOIN tools t ON t.id = s.tool_id
LEFT JOIN tool_serials os ON os.id = s.old_serial_id
JOIN tool_serials ns ON ns.id = s.new_serial_id
WHERE s.production_order_operation_id = $1 AND s.tool_id = $2
ORDER BY s.substituted_at DESC`

type ListToolSerialSubstitutionsParams struct {
	OperationID int64
	ToolID      int64
}

func (q *Queries) ListToolSerialSubstitutions(ctx context.Context, arg ListToolSerialSubstitutionsParams) ([]DBToolSerialSubstitution, error) {
	rows, err := q.db.Query(ctx, listToolSerialSubstitutions, arg.OperationID, arg.ToolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBToolSerialSubstitution
	for rows.Next() {
		var i DBToolSerialSubstitution
		if err := rows.Scan(&i.ID, &i.OperationID, &i.ToolID, &i.ToolCode, &i.ToolName,
			&i.OldSerialID, &i.OldSerialNumber, &i.NewSerialID, &i.NewSerialNumber,
			&i.Reason, &i.SubstitutedAt, &i.SubstitutedBy); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
