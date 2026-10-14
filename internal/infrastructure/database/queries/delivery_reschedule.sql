-- Reprogramações pertencem à empresa do pedido de venda (convenção
-- enterprise_code, a mesma de sales_orders).

-- name: CreateDeliveryReschedule :one
INSERT INTO delivery_reschedules (
 sales_order_code,
 item_code,
 old_date,
 new_date,
 reason,
 created_by,
 enterprise_code)
SELECT sqlc.arg(sales_order_code),
       sqlc.arg(item_code),
       sqlc.arg(old_date),
       sqlc.arg(new_date),
       sqlc.arg(reason),
       sqlc.arg(created_by),
       sqlc.arg(enterprise_code)
 WHERE EXISTS (SELECT 1 FROM sales_orders so
                WHERE so.code = sqlc.arg(sales_order_code) AND so.enterprise_code = sqlc.arg(enterprise_code))
RETURNING *;

-- name: GetDeliveryRescheduleByCode :one
SELECT * FROM delivery_reschedules WHERE code = $1 AND enterprise_code = $2;

-- name: ListReschedulesByOrder :many
SELECT * FROM delivery_reschedules WHERE sales_order_code = $1 AND enterprise_code = $2 ORDER BY created_at DESC;

-- name: ListReschedulesByItem :many
SELECT * FROM delivery_reschedules WHERE item_code = $1 AND enterprise_code = $2 ORDER BY created_at DESC;

-- name: DeleteDeliveryReschedule :execrows
DELETE FROM delivery_reschedules WHERE code = $1 AND enterprise_code = $2;
