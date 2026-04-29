-- name: UpsertDeliveryPromiseParams :one
INSERT INTO delivery_promise_params (use_delivery_promise, blocked_orders_in_promise, default_order_sort,
                                     show_order_values, blocked_export_in_promise, break_tank_occupation, recalculate_after_release,
                                     reprogram_loaded_orders, allow_delivery_date_change, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    ON CONFLICT DO NOTHING
RETURNING *;

-- name: GetDeliveryPromiseParams :one
SELECT * FROM delivery_promise_params ORDER BY id DESC LIMIT 1;

-- name: UpdateDeliveryPromiseParams :one
UPDATE delivery_promise_params
SET use_delivery_promise = $1,
    blocked_orders_in_promise = $2,
    default_order_sort = $3,
    show_order_values = $4,
    blocked_export_in_promise = $5,
    break_tank_occupation = $6,
    recalculate_after_release = $7,
    reprogram_loaded_orders = $8,
    allow_delivery_date_change = $9,
    updated_at = NOW(),
    updated_by = $10
WHERE id = $11
    RETURNING *;
