-- name: CreateMRPItemProfile :one
INSERT INTO mrp_item_profiles (item_code, plan_code, calculation_date, demand, orders_planned, orders_firm, stock_projected, llc, need_date)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    RETURNING *;

-- name: GetMRPItemProfiles :many
SELECT * FROM mrp_item_profiles WHERE item_code = $1 AND plan_code = $2 ORDER BY need_date;

-- name: DeleteProfilesByPlan :exec
DELETE FROM mrp_item_profiles WHERE plan_code = $1;

-- name: StartMRPCalculation :one
INSERT INTO mrp_calculation_logs (plan_code, status, started_at)
VALUES ($1, 'RUNNING', NOW())
    RETURNING *;

-- name: FinishMRPCalculation :one
UPDATE mrp_calculation_logs
SET finished_at = NOW(),
    status = $1,
    errors = $2,
    total_items = $3,
    total_orders = $4
WHERE code = $5
    RETURNING *;

-- name: GetMRPCalculationLog :one
SELECT * FROM mrp_calculation_logs WHERE code = $1;

-- name: ListMRPCalculationLogsByPlan :many
SELECT * FROM mrp_calculation_logs WHERE plan_code = $1 ORDER BY started_at DESC;

-- name: CreateStockSnapshot :one
INSERT INTO stock_snapshots (item_code, warehouse_code, quantity, reserved_qty, safety_stock, snapshot_date)
VALUES ($1, $2, $3, $4, $5, NOW())
    RETURNING *;

-- name: GetStockSnapshot :one
SELECT * FROM stock_snapshots WHERE item_code = $1 ORDER BY snapshot_date DESC LIMIT 1;

-- name: CreateSalesOrderDemand :one
INSERT INTO sales_order_demands (sales_order_code, item_code, mask, quantity, delivered_qty, delivery_date, division_code, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING *;

-- name: GetSalesOrderDemandByCode :one
SELECT * FROM sales_order_demands WHERE code = $1;

-- name: ListSalesOrderDemandsByItem :many
SELECT * FROM sales_order_demands WHERE item_code = $1 AND is_active = TRUE ORDER BY delivery_date;

-- name: UpdateSalesOrderDemandStatus :one
UPDATE sales_order_demands SET status = $1, delivered_qty = $2, updated_at = NOW() WHERE code = $3 RETURNING *;

-- name: CreateConfiguredItemRule :one
INSERT INTO configured_item_rules (item_code, table_type, field_name, rule_type, rule_value, sequence, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING *;

-- name: GetConfiguredItemRules :many
SELECT * FROM configured_item_rules WHERE item_code = $1 AND is_active = TRUE ORDER BY sequence;

-- name: DeleteConfiguredItemRule :exec
UPDATE configured_item_rules SET is_active = FALSE, updated_at = NOW() WHERE code = $1;

-- name: CreateMRPPlannedSuggestion :one
INSERT INTO mrp_planned_suggestions (plan_code, item_code, quantity, need_date, start_date, order_type, demand_type, parent_item_code, llc)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    RETURNING *;

-- name: ListMRPPlannedSuggestions :many
SELECT * FROM mrp_planned_suggestions WHERE plan_code = $1 ORDER BY llc, need_date;

-- name: DeleteMRPPlannedSuggestions :exec
DELETE FROM mrp_planned_suggestions WHERE plan_code = $1;
