-- name: CreateMRPItemProfile :one
INSERT INTO mrp_item_profiles (item_code, plan_code, calculation_date, demand, orders_planned, orders_firm, stock_projected, llc, need_date, enterprise_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, @enterprise_id)
    RETURNING *;

-- name: GetMRPItemProfiles :many
SELECT * FROM mrp_item_profiles WHERE item_code = $1 AND plan_code = $2 AND enterprise_id = @enterprise_id ORDER BY need_date;

-- name: DeleteProfilesByPlan :exec
DELETE FROM mrp_item_profiles WHERE plan_code = $1 AND enterprise_id = @enterprise_id;

-- name: CreateMRPProfileDetail :exec
INSERT INTO mrp_profile_details
    (enterprise_id, plan_code, item_code, need_date, detail_type, source_code, parent_item_code, quantity)
VALUES (@enterprise_id, $1, $2, $3, $4, $5, $6, $7);

-- name: DeleteMRPProfileDetailsByPlan :exec
DELETE FROM mrp_profile_details WHERE plan_code = $1 AND enterprise_id = @enterprise_id;

-- name: StartMRPCalculation :one
INSERT INTO mrp_calculation_logs (plan_code, status, started_at, enterprise_id)
VALUES ($1, 'RUNNING', NOW(), @enterprise_id)
    RETURNING *;

-- name: FinishMRPCalculation :one
UPDATE mrp_calculation_logs
SET finished_at = NOW(),
    status = $1,
    errors = $2,
    total_items = $3,
    total_orders = $4
WHERE code = $5 AND enterprise_id = @enterprise_id
    RETURNING *;

-- name: GetMRPCalculationLog :one
SELECT * FROM mrp_calculation_logs WHERE code = $1 AND enterprise_id = @enterprise_id;

-- name: ListMRPCalculationLogsByPlan :many
SELECT * FROM mrp_calculation_logs WHERE plan_code = $1 AND enterprise_id = @enterprise_id ORDER BY started_at DESC;

-- name: CreateStockSnapshot :one
INSERT INTO stock_snapshots (item_code, warehouse_code, quantity, reserved_qty, safety_stock, snapshot_date, enterprise_id)
VALUES ($1, $2, $3, $4, $5, NOW(), @enterprise_id)
    RETURNING *;

-- name: GetStockSnapshot :one
SELECT * FROM stock_snapshots WHERE item_code = $1 AND enterprise_id = @enterprise_id ORDER BY snapshot_date DESC LIMIT 1;

-- name: CreateSalesOrderDemand :one
INSERT INTO sales_order_demands (sales_order_code, item_code, mask, quantity, delivered_qty, delivery_date, division_code, status, enterprise_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, @enterprise_id)
    RETURNING *;

-- name: GetSalesOrderDemandByCode :one
SELECT * FROM sales_order_demands WHERE code = $1 AND enterprise_id = @enterprise_id;

-- name: ListSalesOrderDemandsByItem :many
SELECT * FROM sales_order_demands WHERE item_code = $1 AND enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY delivery_date;

-- name: UpdateSalesOrderDemandStatus :one
UPDATE sales_order_demands SET status = $1, delivered_qty = $2, updated_at = NOW() WHERE code = $3 AND enterprise_id = @enterprise_id RETURNING *;

-- name: CreateConfiguredItemRule :one
INSERT INTO configured_item_rules (item_code, table_type, field_name, rule_type, rule_value, sequence, created_by, enterprise_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, @enterprise_id)
    RETURNING *;

-- name: GetConfiguredItemRules :many
SELECT * FROM configured_item_rules WHERE item_code = $1 AND enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY sequence;

-- name: DeleteConfiguredItemRule :exec
UPDATE configured_item_rules SET is_active = FALSE, updated_at = NOW() WHERE code = $1 AND enterprise_id = @enterprise_id;

-- name: CreateMRPPlannedSuggestion :one
INSERT INTO mrp_planned_suggestions (order_number, plan_code, item_code, mask, quantity, need_date, start_date, order_type, demand_type, parent_item_code, llc, warehouse_code, inter_factory, source_enterprise_code, auto_release, route_operation_id, operation_id, supplier_code, service_item_code, remittance_type, enterprise_id)
SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, @enterprise_id
WHERE NOT EXISTS (
    SELECT 1 FROM planned_orders po
    WHERE po.enterprise_id = @enterprise_id AND po.order_number = $1 AND po.is_active = TRUE
)
    RETURNING *;

-- name: ListMRPPlannedSuggestions :many
SELECT * FROM mrp_planned_suggestions WHERE plan_code = $1 AND enterprise_id = @enterprise_id ORDER BY llc, need_date;

-- name: DeleteMRPPlannedSuggestions :exec
DELETE FROM mrp_planned_suggestions WHERE plan_code = $1 AND enterprise_id = @enterprise_id;

-- name: UpdateItemPlanningLLCs :exec
UPDATE items i
SET planning_llc = levels.llc
FROM unnest(sqlc.arg(item_codes)::bigint[]) WITH ORDINALITY AS codes(item_code, position)
JOIN unnest(sqlc.arg(llcs)::integer[]) WITH ORDINALITY AS levels(llc, position)
  USING (position)
WHERE i.id = codes.item_code;

-- name: CreateMRPExceptionMessage :one
INSERT INTO mrp_exception_messages
    (plan_code, item_code, message_type, source_code, source_type, description, enterprise_id)
VALUES ($1, $2, $3, $4, $5, $6, @enterprise_id)
RETURNING *;

-- name: ListMRPExceptionMessages :many
SELECT * FROM mrp_exception_messages
WHERE plan_code = $1 AND enterprise_id = @enterprise_id
ORDER BY item_code, code;

-- name: DeleteMRPExceptionMessages :exec
DELETE FROM mrp_exception_messages WHERE plan_code = $1 AND enterprise_id = @enterprise_id;

-- name: ListLatestStockSnapshots :many
SELECT DISTINCT ON (item_code) *
FROM stock_snapshots
WHERE enterprise_id = @enterprise_id
ORDER BY item_code, snapshot_date DESC;

-- name: ListAllActiveConfiguredRules :many
SELECT * FROM configured_item_rules
WHERE enterprise_id = @enterprise_id AND is_active = TRUE
ORDER BY item_code, sequence;
