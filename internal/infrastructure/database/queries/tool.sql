-- ─── tools (master) ────────────────────────────────────────────────────────────

-- name: CreateTool :one
INSERT INTO tools (
    code, name, tool_type, life_type, life_limit, cost, status, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateTool :one
UPDATE tools SET
    name       = $2,
    tool_type  = $3,
    life_type  = $4,
    life_limit = $5,
    cost       = $6,
    status     = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetTool :one
SELECT * FROM tools WHERE id = $1;

-- name: ListTools :many
SELECT * FROM tools
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextToolCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::BIGINT AS next_code FROM tools;

-- name: DeactivateTool :exec
UPDATE tools SET is_active = FALSE, updated_at = NOW() WHERE id = $1;

-- name: ConsumeToolLife :one
UPDATE tools SET life_used = life_used + $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ResetToolLife :one
UPDATE tools SET life_used = 0, status = 'ATIVA', updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListToolsNeedingReplacement :many
SELECT * FROM tools
WHERE is_active = TRUE AND life_limit > 0 AND life_used >= life_limit
ORDER BY code;

-- ─── route_operation_tools (association) ─────────────────────────────────────────

-- name: AddRouteOpTool :one
INSERT INTO route_operation_tools (route_operation_id, tool_id, qty_required)
VALUES ($1, $2, $3)
RETURNING *;

-- name: RemoveRouteOpTool :exec
DELETE FROM route_operation_tools WHERE id = $1;

-- name: ListToolsByRouteOp :many
SELECT rot.id, rot.route_operation_id, rot.tool_id, rot.qty_required,
       t.code AS tool_code, t.name AS tool_name, t.life_type, t.life_limit,
       t.life_used, t.status
FROM route_operation_tools rot
JOIN tools t ON t.id = rot.tool_id
WHERE rot.route_operation_id = $1
ORDER BY rot.id;

-- name: ListToolsByRoute :many
SELECT rot.id, rot.route_operation_id, rot.tool_id, rot.qty_required,
       t.code AS tool_code, t.name AS tool_name, t.life_type, t.life_limit,
       t.life_used, t.status
FROM route_operation_tools rot
JOIN route_operations ro ON ro.id = rot.route_operation_id
JOIN tools t ON t.id = rot.tool_id
WHERE ro.route_id = $1
ORDER BY rot.route_operation_id, rot.id;
