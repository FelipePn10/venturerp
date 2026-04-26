-- name: CreateAllocationBase :one
INSERT INTO allocation_bases (code, description, period, observation, created_by)
VALUES ($1, $2, $3, $4, $5)
    RETURNING *;

-- name: AddAllocationBaseItem :one
INSERT INTO allocation_base_items (allocation_base_id, cost_center_id, amount, percentage)
VALUES ($1, $2, $3, $4)
    RETURNING *;

-- name: GetAllocationBaseByID :one
SELECT * FROM allocation_bases WHERE id = $1;

-- name: GetAllocationBaseItems :many
SELECT * FROM allocation_base_items WHERE allocation_base_id = $1;

-- name: ListAllocationBases :many
SELECT * FROM allocation_bases ORDER BY created_at DESC;

-- name: DeleteAllocationBase :exec
DELETE FROM allocation_bases WHERE id = $1;

-- name: DeleteAllocationBaseItems :exec
DELETE FROM allocation_base_items WHERE allocation_base_id = $1;
