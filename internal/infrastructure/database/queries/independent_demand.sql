-- name: CreateIndependentDemand :one
INSERT INTO independent_demands (code, item_code, mask, cost_center_code, quantity, demand_date, created_by, enterprise_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, @enterprise_id)
    RETURNING *;

-- name: UpdateIndependentDemand :one
UPDATE independent_demands
SET item_code = $1,
    mask = $2,
    cost_center_code = $3,
    quantity = $4,
    demand_date = $5,
    updated_at = NOW()
WHERE code = $6 AND enterprise_id = @enterprise_id
    RETURNING *;

-- name: GetIndependentDemandByCode :one
SELECT * FROM independent_demands WHERE code = $1 AND enterprise_id = @enterprise_id;

-- name: ListIndependentDemands :many
SELECT * FROM independent_demands WHERE enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY demand_date;

-- name: ListDemandsByItem :many
SELECT * FROM independent_demands WHERE item_code = $1 AND enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY demand_date;

-- name: ListDemandsFromDate :many
SELECT * FROM independent_demands WHERE demand_date >= $1 AND enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY demand_date;

-- name: DeleteIndependentDemand :exec
UPDATE independent_demands SET is_active = FALSE, updated_at = NOW() WHERE code = $1 AND enterprise_id = @enterprise_id;
