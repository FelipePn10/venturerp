package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// ─── item_standard_costs ──────────────────────────────────────────────────────

const upsertItemStandardCost = `INSERT INTO item_standard_costs
(item_code, mask, material_cost, labor_cost, overhead_cost, currency, calculated_by)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (item_code, mask) DO UPDATE SET
    material_cost  = EXCLUDED.material_cost,
    labor_cost     = EXCLUDED.labor_cost,
    overhead_cost  = EXCLUDED.overhead_cost,
    currency       = EXCLUDED.currency,
    calculated_at  = NOW(),
    calculated_by  = EXCLUDED.calculated_by
RETURNING id, item_code, mask, material_cost, labor_cost, overhead_cost, total_cost, currency, calculated_at, calculated_by`

type UpsertItemStandardCostParams struct {
	ItemCode     int64
	Mask         string
	MaterialCost pgtype.Numeric
	LaborCost    pgtype.Numeric
	OverheadCost pgtype.Numeric
	Currency     string
	CalculatedBy pgtype.UUID
}

type DBItemStandardCost struct {
	ID           int64
	ItemCode     int64
	Mask         string
	MaterialCost pgtype.Numeric
	LaborCost    pgtype.Numeric
	OverheadCost pgtype.Numeric
	TotalCost    pgtype.Numeric
	Currency     string
	CalculatedAt pgtype.Timestamptz
	CalculatedBy pgtype.UUID
}

func (q *Queries) UpsertItemStandardCost(ctx context.Context, arg UpsertItemStandardCostParams) (DBItemStandardCost, error) {
	row := q.db.QueryRow(ctx, upsertItemStandardCost,
		arg.ItemCode, arg.Mask, arg.MaterialCost, arg.LaborCost, arg.OverheadCost, arg.Currency, arg.CalculatedBy)
	var i DBItemStandardCost
	err := row.Scan(&i.ID, &i.ItemCode, &i.Mask, &i.MaterialCost, &i.LaborCost, &i.OverheadCost, &i.TotalCost, &i.Currency, &i.CalculatedAt, &i.CalculatedBy)
	return i, err
}

const getItemStandardCost = `SELECT id, item_code, mask, material_cost, labor_cost, overhead_cost, total_cost, currency, calculated_at, calculated_by FROM item_standard_costs WHERE item_code=$1 AND mask=$2`

func (q *Queries) GetItemStandardCost(ctx context.Context, itemCode int64, mask string) (DBItemStandardCost, error) {
	row := q.db.QueryRow(ctx, getItemStandardCost, itemCode, mask)
	var i DBItemStandardCost
	err := row.Scan(&i.ID, &i.ItemCode, &i.Mask, &i.MaterialCost, &i.LaborCost, &i.OverheadCost, &i.TotalCost, &i.Currency, &i.CalculatedAt, &i.CalculatedBy)
	return i, err
}

const listItemStandardCosts = `SELECT id, item_code, mask, material_cost, labor_cost, overhead_cost, total_cost, currency, calculated_at, calculated_by FROM item_standard_costs WHERE item_code=$1 ORDER BY mask`

func (q *Queries) ListItemStandardCosts(ctx context.Context, itemCode int64) ([]DBItemStandardCost, error) {
	rows, err := q.db.Query(ctx, listItemStandardCosts, itemCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBItemStandardCost
	for rows.Next() {
		var i DBItemStandardCost
		if err := rows.Scan(&i.ID, &i.ItemCode, &i.Mask, &i.MaterialCost, &i.LaborCost, &i.OverheadCost, &i.TotalCost, &i.Currency, &i.CalculatedAt, &i.CalculatedBy); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ─── work_center_costs ────────────────────────────────────────────────────────

const wccColumns = `id, work_center_id, cost_per_hour, COALESCE(machine_cost_per_hour, cost_per_hour), COALESCE(labor_cost_per_hour, 0), currency, updated_at, updated_by`

const upsertWorkCenterCost = `INSERT INTO work_center_costs (work_center_id, cost_per_hour, machine_cost_per_hour, labor_cost_per_hour, currency, updated_by)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (work_center_id) DO UPDATE SET
    cost_per_hour         = EXCLUDED.cost_per_hour,
    machine_cost_per_hour = EXCLUDED.machine_cost_per_hour,
    labor_cost_per_hour   = EXCLUDED.labor_cost_per_hour,
    currency              = EXCLUDED.currency,
    updated_at            = NOW(),
    updated_by            = EXCLUDED.updated_by
RETURNING id, work_center_id, cost_per_hour, COALESCE(machine_cost_per_hour, cost_per_hour), COALESCE(labor_cost_per_hour, 0), currency, updated_at, updated_by`

type UpsertWorkCenterCostParams struct {
	WorkCenterID       int64
	CostPerHour        pgtype.Numeric
	MachineCostPerHour pgtype.Numeric
	LaborCostPerHour   pgtype.Numeric
	Currency           string
	UpdatedBy          pgtype.UUID
}

type DBWorkCenterCost struct {
	ID                 int64
	WorkCenterID       int64
	CostPerHour        pgtype.Numeric
	MachineCostPerHour pgtype.Numeric
	LaborCostPerHour   pgtype.Numeric
	Currency           string
	UpdatedAt          pgtype.Timestamptz
	UpdatedBy          pgtype.UUID
}

func (q *Queries) UpsertWorkCenterCost(ctx context.Context, arg UpsertWorkCenterCostParams) (DBWorkCenterCost, error) {
	row := q.db.QueryRow(ctx, upsertWorkCenterCost,
		arg.WorkCenterID, arg.CostPerHour, arg.MachineCostPerHour, arg.LaborCostPerHour, arg.Currency, arg.UpdatedBy)
	var i DBWorkCenterCost
	err := row.Scan(&i.ID, &i.WorkCenterID, &i.CostPerHour, &i.MachineCostPerHour, &i.LaborCostPerHour, &i.Currency, &i.UpdatedAt, &i.UpdatedBy)
	return i, err
}

const getWorkCenterCost = `SELECT ` + wccColumns + ` FROM work_center_costs WHERE work_center_id=$1`

func (q *Queries) GetWorkCenterCost(ctx context.Context, workCenterID int64) (DBWorkCenterCost, error) {
	row := q.db.QueryRow(ctx, getWorkCenterCost, workCenterID)
	var i DBWorkCenterCost
	err := row.Scan(&i.ID, &i.WorkCenterID, &i.CostPerHour, &i.MachineCostPerHour, &i.LaborCostPerHour, &i.Currency, &i.UpdatedAt, &i.UpdatedBy)
	return i, err
}

const listWorkCenterCosts = `SELECT ` + wccColumns + ` FROM work_center_costs ORDER BY work_center_id`

func (q *Queries) ListWorkCenterCosts(ctx context.Context) ([]DBWorkCenterCost, error) {
	rows, err := q.db.Query(ctx, listWorkCenterCosts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DBWorkCenterCost
	for rows.Next() {
		var i DBWorkCenterCost
		if err := rows.Scan(&i.ID, &i.WorkCenterID, &i.CostPerHour, &i.MachineCostPerHour, &i.LaborCostPerHour, &i.Currency, &i.UpdatedAt, &i.UpdatedBy); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ─── item_purchase_costs ──────────────────────────────────────────────────────

const upsertItemPurchaseCost = `INSERT INTO item_purchase_costs (item_code, unit_cost, currency, updated_by)
VALUES ($1,$2,$3,$4)
ON CONFLICT (item_code) DO UPDATE SET
    unit_cost  = EXCLUDED.unit_cost,
    currency   = EXCLUDED.currency,
    updated_at = NOW(),
    updated_by = EXCLUDED.updated_by
RETURNING id, item_code, unit_cost, currency, updated_at, updated_by`

type UpsertItemPurchaseCostParams struct {
	ItemCode  int64
	UnitCost  pgtype.Numeric
	Currency  string
	UpdatedBy pgtype.UUID
}

type DBItemPurchaseCost struct {
	ID        int64
	ItemCode  int64
	UnitCost  pgtype.Numeric
	Currency  string
	UpdatedAt pgtype.Timestamptz
	UpdatedBy pgtype.UUID
}

func (q *Queries) UpsertItemPurchaseCost(ctx context.Context, arg UpsertItemPurchaseCostParams) (DBItemPurchaseCost, error) {
	row := q.db.QueryRow(ctx, upsertItemPurchaseCost, arg.ItemCode, arg.UnitCost, arg.Currency, arg.UpdatedBy)
	var i DBItemPurchaseCost
	err := row.Scan(&i.ID, &i.ItemCode, &i.UnitCost, &i.Currency, &i.UpdatedAt, &i.UpdatedBy)
	return i, err
}

const getItemPurchaseCost = `SELECT id, item_code, unit_cost, currency, updated_at, updated_by FROM item_purchase_costs WHERE item_code=$1`

func (q *Queries) GetItemPurchaseCost(ctx context.Context, itemCode int64) (DBItemPurchaseCost, error) {
	row := q.db.QueryRow(ctx, getItemPurchaseCost, itemCode)
	var i DBItemPurchaseCost
	err := row.Scan(&i.ID, &i.ItemCode, &i.UnitCost, &i.Currency, &i.UpdatedAt, &i.UpdatedBy)
	return i, err
}

// ─── cost_rollup_log ──────────────────────────────────────────────────────────

const insertRollupLog = `INSERT INTO cost_rollup_log (item_code, mask, bom_level, material_cost, labor_cost, overhead_cost)
VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, item_code, mask, bom_level, material_cost, labor_cost, overhead_cost, run_at`

type InsertRollupLogParams struct {
	ItemCode     int64
	Mask         string
	BOMLevel     int32
	MaterialCost pgtype.Numeric
	LaborCost    pgtype.Numeric
	OverheadCost pgtype.Numeric
}

type DBCostRollupLog struct {
	ID           int64
	ItemCode     int64
	Mask         string
	BOMLevel     int32
	MaterialCost pgtype.Numeric
	LaborCost    pgtype.Numeric
	OverheadCost pgtype.Numeric
	RunAt        pgtype.Timestamptz
}

func (q *Queries) InsertRollupLog(ctx context.Context, arg InsertRollupLogParams) (DBCostRollupLog, error) {
	row := q.db.QueryRow(ctx, insertRollupLog,
		arg.ItemCode, arg.Mask, arg.BOMLevel, arg.MaterialCost, arg.LaborCost, arg.OverheadCost)
	var i DBCostRollupLog
	err := row.Scan(&i.ID, &i.ItemCode, &i.Mask, &i.BOMLevel, &i.MaterialCost, &i.LaborCost, &i.OverheadCost, &i.RunAt)
	return i, err
}

// ─── routing hours for cost rollup ────────────────────────────────────────────

// GetRouteHoursForItem returns the sum of (setup_time + standard_time) for all active route operations
// of the standard route for a given item + mask.  Returns 0 if no route exists.
const getRouteHoursForItem = `
SELECT COALESCE(SUM(
    COALESCE(ro.setup_time, o.setup_time, 0) +
    COALESCE(ro.standard_time, o.standard_time, 0)
), 0)
FROM manufacturing_routes mr
JOIN route_operations ro ON ro.route_id = mr.id AND ro.is_active = TRUE
JOIN operations o ON o.id = ro.operation_id
WHERE mr.item_code = $1
  AND mr.mask = COALESCE(NULLIF($2,''), mr.mask)
  AND mr.is_standard = TRUE
  AND mr.is_active = TRUE
LIMIT 1`

func (q *Queries) GetRouteHoursForItem(ctx context.Context, itemCode int64, mask string) (float64, error) {
	row := q.db.QueryRow(ctx, getRouteHoursForItem, itemCode, mask)
	var h float64
	return h, row.Scan(&h)
}
