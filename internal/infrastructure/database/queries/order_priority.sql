-- name: CreateOrderPriority :one
INSERT INTO order_priorities (code, interval_start, interval_end, priority, description, created_by, enterprise_id)
VALUES ($1, $2, $3, $4, $5, $6, @enterprise_id)
    RETURNING *;

-- name: UpdateOrderPriority :one
UPDATE order_priorities
SET interval_start = $1,
    interval_end = $2,
    priority = $3,
    description = $4,
    updated_at = NOW()
WHERE code = $5 AND enterprise_id = @enterprise_id
    RETURNING *;

-- name: GetOrderPriorityByCode :one
SELECT * FROM order_priorities WHERE code = $1 AND enterprise_id = @enterprise_id;

-- name: FindPriorityByValue :one
SELECT * FROM order_priorities
WHERE $1 >= interval_start AND $1 < interval_end AND enterprise_id = @enterprise_id AND is_active = TRUE
ORDER BY interval_start
    LIMIT 1;

-- name: ListOrderPriorities :many
SELECT * FROM order_priorities WHERE enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY interval_start;

-- name: DeleteOrderPriority :exec
UPDATE order_priorities SET is_active = FALSE, updated_at = NOW() WHERE code = $1 AND enterprise_id = @enterprise_id;
