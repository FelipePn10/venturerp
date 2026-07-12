-- name: GetPlanningParamByNumber :one
SELECT * FROM planning_params WHERE param_number = $1 AND enterprise_id = @enterprise_id;

-- name: GetPlanningParamByKey :one
SELECT * FROM planning_params WHERE param_key = $1 AND enterprise_id = @enterprise_id;

-- name: ListPlanningParams :many
SELECT * FROM planning_params WHERE enterprise_id = @enterprise_id ORDER BY param_number;

-- name: UpdatePlanningParam :one
UPDATE planning_params
SET value      = $2,
    updated_by = $3,
    updated_at = NOW()
WHERE param_number = $1 AND enterprise_id = @enterprise_id
RETURNING *;
