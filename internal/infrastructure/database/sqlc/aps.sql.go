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
WHERE status IN ('OPEN', 'IN_PROGRESS') AND is_active = TRUE AND enterprise_id = $1
ORDER BY 2 ASC, planned_date ASC`

type DBOpenProductionOrder struct {
	ID          int64
	Priority    int32
	PlannedDate pgtype.Timestamptz
}

// GetOpenProductionOrders lista as OFs abertas da empresa: antes o APS de uma
// empresa percorria as ordens de todas.
func (q *Queries) GetOpenProductionOrders(ctx context.Context, enterpriseID int64) ([]DBOpenProductionOrder, error) {
	rows, err := q.db.Query(ctx, getOpenProductionOrders, enterpriseID)
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

// getOrderOperations: as etapas que faltam. A etapa em andamento entra com o
// que falta (previsto + preparação − apontado, sem nova preparação); a etapa em
// terceiro (sem centro) traz o prazo do fornecedor em dias corridos.
const getOrderOperations = `
SELECT op.id, op.sequence, op.work_center_id,
       CASE WHEN op.status = 'IN_PROGRESS' THEN GREATEST(op.planned_hours + op.setup_hours - op.actual_hours, 0) ELSE op.planned_hours END AS planned_hours,
       CASE WHEN op.status = 'IN_PROGRESS' THEN 0 ELSE op.setup_hours END AS setup_hours,
       op.route_operation_id,
       COALESCE(ro.lead_time_days, o.lead_time_days, 0)::int AS lead_days
FROM production_order_operations op
LEFT JOIN route_operations ro ON ro.id = op.route_operation_id
LEFT JOIN operations o ON o.id = ro.operation_id AND o.enterprise_id = op.enterprise_id
WHERE op.production_order_id=$1 AND op.enterprise_id=$2 AND op.status NOT IN ('DONE','SKIPPED')
ORDER BY op.sequence`

type DBOrderOperation struct {
	ID               int64
	Sequence         int32
	WorkCenterID     pgtype.Int8
	PlannedHours     pgtype.Numeric
	SetupHours       pgtype.Numeric
	RouteOperationID pgtype.Int8
	LeadDays         int32
}

// getOrderOperationEdges traduz a rede de precedências do roteiro para os ids
// das operações desta ordem. Só entram arestas cujas duas pontas ainda estão
// abertas na ordem — operação concluída não segura a sucessora.
// A mesma regra de precedência da execução (OperationPredecessorsPending):
// a OF com retrato do roteiro usa as dependências dela; sem retrato, a rede do
// roteiro; sem rede, a SEQUÊNCIA. Antes, roteiro sem rede (o caso comum) não
// tinha aresta nenhuma e o APS punha corte, dobra e pintura no mesmo instante.
const getOrderOperationEdges = `
WITH abertas AS (
  SELECT id, sequence, route_operation_id FROM production_order_operations
  WHERE production_order_id = $1 AND enterprise_id = $2 AND status NOT IN ('DONE','SKIPPED')
), retrato AS (
  SELECT (SELECT routing_snapshot_at IS NOT NULL FROM production_orders WHERE id = $1 AND enterprise_id = $2) AS sim
), rede AS (
  SELECT p.id AS predecessor_id, s.id AS successor_id, COALESCE(n.overlap_pct, 0) AS overlap_pct
  FROM route_operation_network n
  JOIN route_operations a ON a.id = n.predecessor_id AND a.is_active
  JOIN route_operations b ON b.id = n.successor_id AND b.is_active
  JOIN abertas p ON p.route_operation_id = n.predecessor_id
  JOIN abertas s ON s.route_operation_id = n.successor_id
), dependencias AS (
  SELECT d.predecessor_id, d.successor_id, 0::numeric AS overlap_pct
  FROM production_order_operation_dependencies d
  JOIN abertas p ON p.id = d.predecessor_id
  JOIN abertas s ON s.id = d.successor_id
), roteiro_tem_rede AS (
  SELECT EXISTS (SELECT 1 FROM route_operation_network n
                 JOIN route_operations a ON a.id = n.predecessor_id AND a.is_active
                 JOIN route_operations b ON b.id = n.successor_id AND b.is_active
                 JOIN production_order_operations x ON x.route_operation_id = a.id AND x.production_order_id = $1 AND x.enterprise_id = $2) AS sim
), sequencia AS (
  SELECT LAG(id) OVER (ORDER BY sequence, id) AS predecessor_id, id AS successor_id, 0::numeric AS overlap_pct FROM abertas
)
SELECT predecessor_id, successor_id, overlap_pct FROM dependencias WHERE (SELECT sim FROM retrato)
UNION ALL
SELECT predecessor_id, successor_id, overlap_pct FROM rede WHERE NOT (SELECT sim FROM retrato)
UNION ALL
SELECT predecessor_id, successor_id, overlap_pct FROM sequencia
 WHERE predecessor_id IS NOT NULL AND NOT (SELECT sim FROM retrato) AND NOT (SELECT sim FROM roteiro_tem_rede)`

type DBOrderOperationEdge struct {
	PredecessorID int64
	SuccessorID   int64
	OverlapPct    pgtype.Numeric
}

func (q *Queries) GetOrderOperationEdges(ctx context.Context, orderID, enterpriseID int64) ([]DBOrderOperationEdge, error) {
	rows, err := q.db.Query(ctx, getOrderOperationEdges, orderID, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DBOrderOperationEdge{}
	for rows.Next() {
		var e DBOrderOperationEdge
		if err := rows.Scan(&e.PredecessorID, &e.SuccessorID, &e.OverlapPct); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (q *Queries) GetOrderOperations(ctx context.Context, orderID, enterpriseID int64) ([]DBOrderOperation, error) {
	rows, err := q.db.Query(ctx, getOrderOperations, orderID, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBOrderOperation
	for rows.Next() {
		var i DBOrderOperation
		if err := rows.Scan(&i.ID, &i.Sequence, &i.WorkCenterID, &i.PlannedHours, &i.SetupHours, &i.RouteOperationID, &i.LeadDays); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// listSetupMatrix traz as transições de preparação de um centro de trabalho.
const listSetupMatrix = `
SELECT id, work_center_id, from_item_code, to_item_code, from_family, to_family,
       setup_minutes, is_active
FROM setup_matrix
WHERE work_center_id = $1 AND enterprise_id = $2 AND is_active`

type DBSetupTransition struct {
	ID           int64
	WorkCenterID int64
	FromItemCode *int64
	ToItemCode   *int64
	FromFamily   pgtype.Text
	ToFamily     pgtype.Text
	SetupMinutes pgtype.Numeric
	IsActive     bool
}

func (q *Queries) ListSetupMatrix(ctx context.Context, workCenterID, enterpriseID int64) ([]DBSetupTransition, error) {
	rows, err := q.db.Query(ctx, listSetupMatrix, workCenterID, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DBSetupTransition{}
	for rows.Next() {
		var t DBSetupTransition
		if err := rows.Scan(&t.ID, &t.WorkCenterID, &t.FromItemCode, &t.ToItemCode,
			&t.FromFamily, &t.ToFamily, &t.SetupMinutes, &t.IsActive); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// getOrderItem devolve o item da ordem e a família usada pela matriz de setup.
// A família sai da classificação comercial do item — é por ela que a fábrica
// agrupa "mesma cor", "mesma espessura".
const getOrderItem = `
SELECT po.item_code, COALESCE(NULLIF(i.setup_family, ''), '')
FROM production_orders po
LEFT JOIN items i ON i.code = po.item_code AND i.enterprise_id = po.enterprise_id
WHERE po.id = $1`

func (q *Queries) GetOrderItem(ctx context.Context, orderID int64) (int64, string, error) {
	var itemCode int64
	var familia string
	err := q.db.QueryRow(ctx, getOrderItem, orderID).Scan(&itemCode, &familia)
	return itemCode, familia, err
}

// upsertSetupMatrix grava (ou atualiza) uma transição da matriz de preparação.
const upsertSetupMatrix = `
INSERT INTO setup_matrix (enterprise_id, work_center_id, from_item_code, to_item_code,
                          from_family, to_family, setup_minutes, notes, is_active)
VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),$7,NULLIF($8,''),$9)
ON CONFLICT (enterprise_id, work_center_id,
             COALESCE(from_item_code,-1), COALESCE(to_item_code,-1),
             COALESCE(from_family,''), COALESCE(to_family,''))
DO UPDATE SET setup_minutes = EXCLUDED.setup_minutes,
              notes = EXCLUDED.notes,
              is_active = EXCLUDED.is_active,
              updated_at = NOW()
RETURNING id`

func (q *Queries) UpsertSetupMatrix(ctx context.Context, enterpriseID, workCenterID int64,
	fromItem, toItem *int64, fromFamily, toFamily string, minutes float64, notes string, active bool) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, upsertSetupMatrix, enterpriseID, workCenterID, fromItem, toItem,
		fromFamily, toFamily, minutes, notes, active).Scan(&id)
	return id, err
}

const deleteSetupMatrix = `DELETE FROM setup_matrix WHERE id = $1 AND enterprise_id = $2`

func (q *Queries) DeleteSetupMatrix(ctx context.Context, id, enterpriseID int64) (int64, error) {
	tag, err := q.db.Exec(ctx, deleteSetupMatrix, id, enterpriseID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
