-- Parâmetros da promessa de entrega: uma configuração por empresa.

-- name: UpsertDeliveryPromiseParams :one
INSERT INTO delivery_promise_params (use_delivery_promise, blocked_orders_in_promise, default_order_sort,
                                     show_order_values, blocked_export_in_promise, break_tank_occupation, recalculate_after_release,
                                     reprogram_loaded_orders, allow_delivery_date_change, updated_by, enterprise_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    ON CONFLICT (enterprise_id) DO UPDATE SET
    use_delivery_promise = EXCLUDED.use_delivery_promise,
    blocked_orders_in_promise = EXCLUDED.blocked_orders_in_promise,
    default_order_sort = EXCLUDED.default_order_sort,
    show_order_values = EXCLUDED.show_order_values,
    blocked_export_in_promise = EXCLUDED.blocked_export_in_promise,
    break_tank_occupation = EXCLUDED.break_tank_occupation,
    recalculate_after_release = EXCLUDED.recalculate_after_release,
    reprogram_loaded_orders = EXCLUDED.reprogram_loaded_orders,
    allow_delivery_date_change = EXCLUDED.allow_delivery_date_change,
    updated_at = NOW(),
    updated_by = EXCLUDED.updated_by
RETURNING *;

-- name: GetDeliveryPromiseParams :one
SELECT * FROM delivery_promise_params WHERE enterprise_id = $1;
