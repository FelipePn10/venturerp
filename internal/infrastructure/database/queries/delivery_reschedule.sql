-- name: CreateDeliveryReschedule :one
INSERT INTO delivery_reschedules (
 code,
 sales_order_code,
 item_code,
 old_date,
 new_date,
 reason,
 created_by)
VALUES ($1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7)
    RETURNING *;
-- name: GetDeliveryRescheduleByCode :one
SELECT * FROM delivery_reschedules WHERE code = $1;

-- name: ListReschedulesByOrder :many
SELECT * FROM delivery_reschedules WHERE sales_order_code = $1 ORDER BY created_at DESC;

-- name: ListReschedulesByItem :many
SELECT * FROM delivery_reschedules WHERE item_code = $1 ORDER BY created_at DESC;

-- name: DeleteDeliveryReschedule :exec
DELETE FROM delivery_reschedules WHERE code = $1;
