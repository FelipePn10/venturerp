-- ─── operations ──────────────────────────────────────────────────────────────

-- name: CreateOperation :one
INSERT INTO operations (
    code, name, description, origin, situation,
    default_work_center_id, standard_time, setup_time,
    is_active, created_by
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8,
    TRUE, $9
) RETURNING *;

-- name: UpdateOperation :one
UPDATE operations SET
    name = $2,
    description = $3,
    origin = $4,
    situation = $5,
    default_work_center_id = $6,
    standard_time = $7,
    setup_time = $8,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetOperationByID :one
SELECT * FROM operations WHERE id = $1;

-- name: ListOperations :many
SELECT * FROM operations
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: DeactivateOperation :exec
UPDATE operations SET is_active = FALSE, updated_at = NOW() WHERE id = $1;

-- name: NextOperationCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM operations;

-- ─── manufacturing_routes ─────────────────────────────────────────────────────

-- name: CreateRoute :one
INSERT INTO manufacturing_routes (
    code, item_code, mask, alternative, description,
    situation, is_standard, is_active, created_by
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, TRUE, $8
) RETURNING *;

-- name: UpdateRoute :one
UPDATE manufacturing_routes SET
    description = $2,
    situation = $3,
    is_standard = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetRouteByID :one
SELECT * FROM manufacturing_routes WHERE id = $1;

-- name: GetRouteByItemAndAlternative :one
SELECT * FROM manufacturing_routes
WHERE item_code = $1
  AND COALESCE(mask, '') = COALESCE($2, '')
  AND alternative = $3
  AND is_active = TRUE;

-- name: ListRoutesByItem :many
SELECT * FROM manufacturing_routes
WHERE item_code = $1 AND is_active = TRUE
ORDER BY alternative;

-- name: DeactivateRoute :exec
UPDATE manufacturing_routes SET is_active = FALSE, updated_at = NOW() WHERE id = $1;

-- name: NextRouteCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM manufacturing_routes;

-- name: ItemHasRoute :one
SELECT EXISTS(
    SELECT 1 FROM manufacturing_routes
    WHERE item_code = $1 AND is_active = TRUE
) AS has_route;

-- name: GetStandardRouteForItem :one
SELECT * FROM manufacturing_routes
WHERE item_code = $1
  AND COALESCE(mask, '') = COALESCE($2, '')
  AND is_active = TRUE
ORDER BY is_standard DESC, alternative
LIMIT 1;

-- ─── route_operations ────────────────────────────────────────────────────────

-- name: AddRouteOperation :one
INSERT INTO route_operations (
    route_id, sequence, operation_id, work_center_id,
    standard_time, setup_time, situation, notes, is_active
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8, TRUE
) RETURNING *;

-- name: UpdateRouteOperation :one
UPDATE route_operations SET
    work_center_id = $2,
    standard_time = $3,
    setup_time = $4,
    situation = $5,
    notes = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetRouteOperations :many
SELECT
    ro.*,
    op.name AS operation_name,
    op.origin AS operation_origin,
    op.standard_time AS op_standard_time,
    op.setup_time AS op_setup_time,
    mt.name AS work_center_name,
    COALESCE(mt.requires_operator, TRUE) AS requires_operator
FROM route_operations ro
JOIN operations op ON op.id = ro.operation_id
LEFT JOIN machine_types mt ON mt.id = COALESCE(ro.work_center_id, op.default_work_center_id)
WHERE ro.route_id = $1 AND ro.is_active = TRUE
ORDER BY ro.sequence;

-- name: RemoveRouteOperation :exec
UPDATE route_operations SET is_active = FALSE, updated_at = NOW() WHERE id = $1;

-- ─── route_operation_network ─────────────────────────────────────────────────

-- name: UpsertNetworkEdge :one
INSERT INTO route_operation_network (predecessor_id, successor_id, overlap_pct)
VALUES ($1, $2, $3)
ON CONFLICT (predecessor_id, successor_id)
DO UPDATE SET overlap_pct = EXCLUDED.overlap_pct
RETURNING *;

-- name: DeleteNetworkEdge :exec
DELETE FROM route_operation_network
WHERE predecessor_id = $1 AND successor_id = $2;

-- name: GetNetworkEdges :many
SELECT ron.*
FROM route_operation_network ron
JOIN route_operations ro ON ro.id = ron.predecessor_id
WHERE ro.route_id = $1
ORDER BY ron.predecessor_id, ron.successor_id;

-- name: GetExternalRouteOpsForItem :many
SELECT
    ro.id,
    ro.operation_id,
    ro.work_center_id,
    op.name AS operation_name,
    op.origin,
    COALESCE(ro.standard_time, op.standard_time) AS effective_hours
FROM manufacturing_routes mr
JOIN route_operations ro ON ro.route_id = mr.id
JOIN operations op ON op.id = ro.operation_id
WHERE mr.item_code = $1
  AND mr.is_active = TRUE
  AND mr.is_standard = TRUE
  AND ro.is_active = TRUE
  AND op.origin IN ('EXTERNA', 'TERCEIROS');
