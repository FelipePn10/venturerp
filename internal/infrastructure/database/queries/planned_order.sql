-- name: CreatePlannedOrder :one
INSERT INTO planned_orders (
    order_number,
    item_code,
    mask,
    quantity,
    quantity_loss,
    quantity_corrected,
    order_type,
    status,
    plan_code,
    demand_type,
    demand_code,
    need_date,
    start_date,
    end_date,
    cost_center_code,
    employee_code,
    machine_code,
    warehouse_code,
    inter_factory,
    source_enterprise_code,
    auto_release,
    mrp_suggestion_code,
    production_time,
    priority,
    llc,
    notes,
    parent_order_code,
    sales_order_code,
    created_by,
    enterprise_id
)
VALUES (
           $1, $2, $3, $4, $5, $6,
           $7, $8, $9, $10, $11, $12,
           $13, $14, $15, $16, $17, $18,
           $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, @enterprise_id
       )
    RETURNING *;

-- name: GetPlannedOrderByCode :one
SELECT * FROM planned_orders WHERE code = $1 AND enterprise_id = @enterprise_id;

-- name: GetPlannedOrderByNumber :one
SELECT * FROM planned_orders WHERE order_number = $1 AND enterprise_id = @enterprise_id;

-- name: GetPlannedOrderByMRPSuggestionCode :one
SELECT * FROM planned_orders WHERE mrp_suggestion_code = $1 AND enterprise_id = @enterprise_id;

-- name: ListPlannedOrders :many
SELECT * FROM planned_orders WHERE enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY need_date;

-- name: ListPlannedOrdersByPlan :many
SELECT * FROM planned_orders WHERE plan_code = $1 AND enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY need_date;

-- name: ListPlannedOrdersByItem :many
SELECT * FROM planned_orders WHERE item_code = $1 AND enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY need_date;

-- name: ListPlannedOrdersByType :many
SELECT * FROM planned_orders WHERE order_type = $1 AND enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY need_date;

-- name: ListPlannedOrdersByStatus :many
SELECT * FROM planned_orders WHERE status = $1 AND enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY need_date;

-- name: UpdatePlannedOrderStatus :one
UPDATE planned_orders SET status = $1, updated_at = NOW() WHERE code = $2 AND enterprise_id = @enterprise_id RETURNING *;

-- name: FirmPlannedOrder :one
UPDATE planned_orders SET is_firm = TRUE, status = 'RELEASED', updated_at = NOW() WHERE code = $1 AND enterprise_id = @enterprise_id RETURNING *;

-- name: SetPlannedOrderPlanningState :one
UPDATE planned_orders
SET status = $1, is_firm = $2, updated_at = NOW()
WHERE code = $3 AND enterprise_id = @enterprise_id
RETURNING *;

-- name: IsPlannedOrderItemKanban :one
SELECT EXISTS (
    SELECT 1 FROM kanban_cards
    WHERE item_code = $1 AND enterprise_id = @enterprise_id AND is_active = TRUE
) AS is_kanban;

-- name: HasPlannedOrderProductionMovements :one
SELECT EXISTS (
    SELECT 1
    FROM production_orders po
    JOIN planned_orders planned ON planned.code = po.planned_order_id
    WHERE po.planned_order_id = $1
      AND planned.enterprise_id = @enterprise_id
      AND (
          EXISTS (SELECT 1 FROM production_appointments pa WHERE pa.production_order_id = po.id)
          OR EXISTS (SELECT 1 FROM production_consumptions pc WHERE pc.production_order_id = po.id)
          OR EXISTS (
              SELECT 1 FROM stock_movements sm
              WHERE sm.reference_type = 'PRODUCTION_ORDER' AND sm.reference_code = po.id
          )
      )
) AS has_movements;

-- name: UpdatePlannedOrderDates :one
UPDATE planned_orders SET start_date = $1, end_date = $2, updated_at = NOW() WHERE code = $3 AND enterprise_id = @enterprise_id RETURNING *;

-- name: DeletePlannedOrder :exec
UPDATE planned_orders SET is_active = FALSE, updated_at = NOW() WHERE code = $1 AND enterprise_id = @enterprise_id;

-- name: GetNextOrderNumber :one
SELECT COALESCE(MAX(order_number), 0) + 1 AS next_number FROM planned_orders WHERE enterprise_id = @enterprise_id;

-- name: DeleteOrdersByPlan :exec
UPDATE planned_orders SET is_active = FALSE, updated_at = NOW() WHERE plan_code = $1 AND enterprise_id = @enterprise_id;

-- name: ListFirmPlannedOrdersByItems :many
SELECT code, item_code, quantity, need_date
FROM planned_orders
WHERE item_code = ANY(@item_codes::bigint[])
  AND enterprise_id = @enterprise_id
  AND is_firm = TRUE
  AND is_active = TRUE
ORDER BY item_code, need_date;
