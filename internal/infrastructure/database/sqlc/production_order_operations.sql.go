package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// ─── production_order_operations ─────────────────────────────────────────────

const createProductionOrderOperation = `INSERT INTO production_order_operations
(production_order_id, route_operation_id, sequence, operation_name, work_center_id,
 planned_hours, setup_hours, planned_qty, status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'PENDING')
RETURNING id, production_order_id, route_operation_id, sequence, operation_name, work_center_id,
          planned_hours, setup_hours, actual_hours, status, started_at, completed_at, notes, created_at, updated_at`

type CreateProductionOrderOperationParams struct {
	ProductionOrderID int64
	RouteOperationID  pgtype.Int8
	Sequence          int16
	OperationName     string
	WorkCenterID      pgtype.Int8
	PlannedHours      pgtype.Numeric
	SetupHours        pgtype.Numeric
	// PlannedQty é quanto precisa ENTRAR nesta etapa para a ordem fechar a
	// quantidade boa. Com refugo ela é maior nas primeiras operações.
	PlannedQty pgtype.Numeric
}

type DBProductionOrderOperation struct {
	ID                int64
	ProductionOrderID int64
	RouteOperationID  pgtype.Int8
	Sequence          int16
	OperationName     string
	WorkCenterID      pgtype.Int8
	PlannedHours      pgtype.Numeric
	SetupHours        pgtype.Numeric
	ActualHours       pgtype.Numeric
	Status            string
	StartedAt         pgtype.Timestamptz
	CompletedAt       pgtype.Timestamptz
	Notes             pgtype.Text
	CreatedAt         pgtype.Timestamptz
	UpdatedAt         pgtype.Timestamptz
}

func (q *Queries) CreateProductionOrderOperation(ctx context.Context, arg CreateProductionOrderOperationParams) (DBProductionOrderOperation, error) {
	row := q.db.QueryRow(ctx, createProductionOrderOperation,
		arg.ProductionOrderID, arg.RouteOperationID, arg.Sequence, arg.OperationName,
		arg.WorkCenterID, arg.PlannedHours, arg.SetupHours, arg.PlannedQty)
	return scanPOO(row)
}

const listProductionOrderOperations = `SELECT id, production_order_id, route_operation_id, sequence,
operation_name, work_center_id, planned_hours, setup_hours, actual_hours, status,
started_at, completed_at, notes, created_at, updated_at
FROM production_order_operations WHERE production_order_id=$1 ORDER BY sequence`

func (q *Queries) ListProductionOrderOperations(ctx context.Context, orderID int64) ([]DBProductionOrderOperation, error) {
	rows, err := q.db.Query(ctx, listProductionOrderOperations, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBProductionOrderOperation
	for rows.Next() {
		poo, err := scanPOORow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, poo)
	}
	return items, rows.Err()
}

const getProductionOrderOperation = `SELECT id, production_order_id, route_operation_id, sequence,
operation_name, work_center_id, planned_hours, setup_hours, actual_hours, status,
started_at, completed_at, notes, created_at, updated_at
FROM production_order_operations WHERE id=$1`

func (q *Queries) GetProductionOrderOperation(ctx context.Context, id int64) (DBProductionOrderOperation, error) {
	row := q.db.QueryRow(ctx, getProductionOrderOperation, id)
	return scanPOO(row)
}

// $2 is cast to ::text at every use so Postgres deduces a single, consistent
// type for the parameter. Without the casts the assignment (status=$2) and the
// comparisons ($2='IN_PROGRESS', $2 IN (...)) let the planner infer conflicting
// types, raising "inconsistent types deduced for parameter $2" (SQLSTATE 42P08).
const advanceProductionOrderOperation = `UPDATE production_order_operations
SET status=$2::text,
    started_at=CASE WHEN $2::text='IN_PROGRESS' THEN NOW() ELSE started_at END,
    completed_at=CASE WHEN $2::text IN ('DONE','SKIPPED') THEN NOW() ELSE completed_at END,
    updated_at=NOW()
WHERE id=$1
RETURNING id, production_order_id, route_operation_id, sequence, operation_name, work_center_id,
          planned_hours, setup_hours, actual_hours, status, started_at, completed_at, notes, created_at, updated_at`

func (q *Queries) AdvanceProductionOrderOperation(ctx context.Context, id int64, status string) (DBProductionOrderOperation, error) {
	row := q.db.QueryRow(ctx, advanceProductionOrderOperation, id, status)
	return scanPOO(row)
}

const addActualHours = `UPDATE production_order_operations SET actual_hours=actual_hours+$2, updated_at=NOW() WHERE id=$1`

func (q *Queries) AddActualHours(ctx context.Context, id int64, hours pgtype.Numeric) error {
	_, err := q.db.Exec(ctx, addActualHours, id, hours)
	return err
}

// ─── helpers for exploding route into order ───────────────────────────────────

// GetRouteOperationsForOrder fetches route operations to explode into production_order_operations.
const getRouteOpsForExplode = `
SELECT ro.id, ro.sequence, o.name, ro.work_center_id,
       COALESCE(ro.standard_time, o.standard_time, 0) AS planned_hours,
       COALESCE(ro.setup_time, o.setup_time, 0) AS setup_hours,
       COALESCE(ro.scrap_pct, o.scrap_pct, 0) AS effective_scrap,
       ro.inspection_required
FROM route_operations ro
JOIN operations o ON o.id = ro.operation_id
WHERE ro.route_id = $1 AND ro.is_active = TRUE
ORDER BY ro.sequence`

type DBRouteOpForExplode struct {
	ID            int64
	Sequence      int16
	OperationName string
	WorkCenterID  pgtype.Int8
	PlannedHours  float64
	SetupHours    float64
	// EffectiveScrap é o refugo que vale na etapa (override ∘ padrão da operação).
	EffectiveScrap float64
	// InspectionRequired marca a etapa como ponto de inspeção.
	InspectionRequired bool
}

func (q *Queries) GetRouteOpsForExplode(ctx context.Context, routeID int64) ([]DBRouteOpForExplode, error) {
	rows, err := q.db.Query(ctx, getRouteOpsForExplode, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBRouteOpForExplode
	for rows.Next() {
		var i DBRouteOpForExplode
		if err := rows.Scan(&i.ID, &i.Sequence, &i.OperationName,
			&i.WorkCenterID, &i.PlannedHours, &i.SetupHours,
			&i.EffectiveScrap, &i.InspectionRequired); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ─── scanners ─────────────────────────────────────────────────────────────────

type pooScanner interface {
	Scan(...any) error
}

func scanPOO(s pooScanner) (DBProductionOrderOperation, error) {
	var i DBProductionOrderOperation
	err := s.Scan(&i.ID, &i.ProductionOrderID, &i.RouteOperationID, &i.Sequence,
		&i.OperationName, &i.WorkCenterID, &i.PlannedHours, &i.SetupHours, &i.ActualHours,
		&i.Status, &i.StartedAt, &i.CompletedAt, &i.Notes, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

func scanPOORow(rows interface{ Scan(...any) error }) (DBProductionOrderOperation, error) {
	return scanPOO(rows)
}

// GetRouteNetworkForExplode devolve as precedências do roteiro. A cascata de
// refugo precisa delas: numa rede que converge, o que sai de uma operação é o
// maior do que seus sucessores pedem, e encadear só por sequência daria outro
// número.
const getRouteNetworkForExplode = `
SELECT ron.predecessor_id, ron.successor_id, ron.overlap_pct
FROM route_operation_network ron
JOIN route_operations ro ON ro.id = ron.predecessor_id
WHERE ro.route_id = $1`

type DBRouteEdgeForExplode struct {
	PredecessorID int64
	SuccessorID   int64
	OverlapPct    float64
}

func (q *Queries) GetRouteNetworkForExplode(ctx context.Context, routeID int64) ([]DBRouteEdgeForExplode, error) {
	rows, err := q.db.Query(ctx, getRouteNetworkForExplode, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBRouteEdgeForExplode
	for rows.Next() {
		var e DBRouteEdgeForExplode
		if err := rows.Scan(&e.PredecessorID, &e.SuccessorID, &e.OverlapPct); err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

// GetProductionOrderQty devolve a quantidade boa que a ordem precisa entregar.
// É o ponto de partida da cascata de refugo: é a partir dela que se descobre
// quanto cada etapa precisa processar.
const getProductionOrderQty = `SELECT planned_qty, item_code FROM production_orders WHERE id = $1`

func (q *Queries) GetProductionOrderQty(ctx context.Context, orderID int64) (float64, int64, error) {
	var qty pgtype.Numeric
	var itemCode int64
	if err := q.db.QueryRow(ctx, getProductionOrderQty, orderID).Scan(&qty, &itemCode); err != nil {
		return 0, 0, err
	}
	f, _ := qty.Float64Value()
	return f.Float64, itemCode, nil
}
