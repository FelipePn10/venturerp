-- name: GetPlanningParamByNumber :one
SELECT * FROM planning_params WHERE param_number = $1;

-- name: GetPlanningParamByKey :one
SELECT * FROM planning_params WHERE param_key = $1;

-- name: ListPlanningParams :many
SELECT * FROM planning_params ORDER BY param_number;

-- name: UpdatePlanningParam :one
UPDATE planning_params
SET value      = $2,
    updated_by = $3,
    updated_at = NOW()
WHERE param_number = $1
RETURNING *;
