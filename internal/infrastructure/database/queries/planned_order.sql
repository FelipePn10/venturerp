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
    production_time,
    priority,
    llc,
    notes,
    parent_order_code,
    sales_order_code,
    created_by
)
VALUES (
           $1, $2, $3, $4, $5, $6,
           $7, $8, $9, $10, $11, $12,
           $13, $14, $15, $16, $17, $18,
           $19, $20, $21, $22, $23, $24
       )
    RETURNING *;

-- name: GetPlannedOrderByCode :one
SELECT * FROM planned_orders WHERE code = $1;

-- name: GetPlannedOrderByNumber :one
SELECT * FROM planned_orders WHERE order_number = $1;

-- name: ListPlannedOrders :many
SELECT * FROM planned_orders WHERE is_active = TRUE ORDER BY need_date;

-- name: ListPlannedOrdersByPlan :many
SELECT * FROM planned_orders WHERE plan_code = $1 AND is_active = TRUE ORDER BY need_date;

-- name: ListPlannedOrdersByItem :many
SELECT * FROM planned_orders WHERE item_code = $1 AND is_active = TRUE ORDER BY need_date;

-- name: ListPlannedOrdersByType :many
SELECT * FROM planned_orders WHERE order_type = $1 AND is_active = TRUE ORDER BY need_date;

-- name: ListPlannedOrdersByStatus :many
SELECT * FROM planned_orders WHERE status = $1 AND is_active = TRUE ORDER BY need_date;

-- name: UpdatePlannedOrderStatus :one
UPDATE planned_orders SET status = $1, updated_at = NOW() WHERE code = $2 RETURNING *;

-- name: FirmPlannedOrder :one
UPDATE planned_orders SET is_firm = TRUE, status = 'RELEASED', updated_at = NOW() WHERE code = $1 RETURNING *;

-- name: UpdatePlannedOrderDates :one
UPDATE planned_orders SET start_date = $1, end_date = $2, updated_at = NOW() WHERE code = $3 RETURNING *;

-- name: DeletePlannedOrder :exec
UPDATE planned_orders SET is_active = FALSE, updated_at = NOW() WHERE code = $1;

-- name: GetNextOrderNumber :one
SELECT COALESCE(MAX(order_number), 0) + 1 AS next_number FROM planned_orders;

-- name: DeleteOrdersByPlan :exec
UPDATE planned_orders SET is_active = FALSE, updated_at = NOW() WHERE plan_code = $1;
