-- name: CreateAllocationBase :one
INSERT INTO allocation_bases (code, description, period, observation, created_by, enterprise_id)
VALUES ($1, $2, $3, $4, $5, sqlc.arg(enterprise_id))
    RETURNING *;

-- name: AddAllocationBaseItem :one
INSERT INTO allocation_base_items (allocation_base_code, cost_center_code, amount, percentage)
SELECT $1, $2, $3, $4
FROM allocation_bases b
WHERE b.code = $1 AND b.enterprise_id = sqlc.arg(enterprise_id)
    RETURNING *;

-- name: GetAllocationBaseByCode :one
SELECT * FROM allocation_bases WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- As linhas seguem a base: `allocation_base_items` não tem coluna de empresa, e
-- a posse dela é a da base de rateio referenciada.
-- name: GetAllocationBaseItems :many
SELECT i.* FROM allocation_base_items i
JOIN allocation_bases b ON b.code = i.allocation_base_code AND b.enterprise_id = sqlc.arg(enterprise_id)
WHERE i.allocation_base_code = $1;

-- name: ListAllocationBases :many
SELECT * FROM allocation_bases WHERE enterprise_id = sqlc.arg(enterprise_id) ORDER BY created_at DESC;

-- name: DeleteAllocationBase :exec
DELETE FROM allocation_bases WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: DeleteAllocationBaseItems :exec
DELETE FROM allocation_base_items i
USING allocation_bases b
WHERE i.allocation_base_code = $1
  AND b.code = i.allocation_base_code
  AND b.enterprise_id = sqlc.arg(enterprise_id);
