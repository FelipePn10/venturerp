-- name: CreateCostCenter :one
INSERT INTO cost_centers (code, description, parent_code, type, is_ratio, start_date, end_date, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING *;

-- name: UpdateCostCenter :one
UPDATE cost_centers
SET description = $1,
    parent_code = $2,
    type = $3,
    is_ratio = $4,
    start_date = $5,
    end_date = $6,
    updated_at = NOW()
WHERE id = $7
    RETURNING *;

-- name: GetCostCenterByCode :one
SELECT * FROM cost_centers WHERE code = $1;

-- name: ListCostCenters :many
SELECT * FROM cost_centers WHERE is_active = TRUE ORDER BY code;

-- name: ListCostCentersByType :many
SELECT * FROM cost_centers WHERE type = $1 AND is_active = TRUE ORDER BY code;

-- name: DeleteCostCenter :exec
UPDATE cost_centers SET is_active = FALSE, updated_at = NOW() WHERE code = $1;
