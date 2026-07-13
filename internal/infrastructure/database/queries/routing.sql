-- ─── operations ──────────────────────────────────────────────────────────────

-- name: CreateOperation :one
INSERT INTO operations (
    code, name, description, origin, situation,
    default_work_center_id, standard_time, setup_time,
    run_time, labor_time, run_time_base_qty,
    queue_time, wait_time, move_time, crew_size, time_unit,
    supplier_id, service_item_code, cost_per_unit, lead_time_days, third_party_remittance,
    is_active, created_by
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8,
    $9, $10, $11,
    $12, $13, $14, $15, $16,
    $17, $18, $19, $20, $21,
    TRUE, $22
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
    run_time = $9,
    labor_time = $10,
    run_time_base_qty = $11,
    queue_time = $12,
    wait_time = $13,
    move_time = $14,
    crew_size = $15,
    time_unit = $16,
    supplier_id = $17,
    service_item_code = $18,
    cost_per_unit = $19,
    lead_time_days = $20,
    third_party_remittance = $21,
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

-- name: OperationUsedInRoutes :one
SELECT EXISTS(SELECT 1 FROM route_operations WHERE operation_id = $1 AND is_active);

-- name: NextOperationCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::BIGINT AS next_code FROM operations;

-- ─── manufacturing_routes ─────────────────────────────────────────────────────

-- name: CreateRoute :one
INSERT INTO manufacturing_routes (
    code, item_code, mask, alternative, description,
    situation, is_standard, valid_from, valid_to, is_active, created_by
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, TRUE, $10
) RETURNING *;

-- name: UpdateRoute :one
UPDATE manufacturing_routes SET
    description = $2,
    situation = $3,
    is_standard = $4,
    valid_from = $5,
    valid_to = $6,
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
SELECT (COALESCE(MAX(code), 0) + 1)::BIGINT AS next_code FROM manufacturing_routes;

-- name: ItemHasRoute :one
SELECT EXISTS(
    SELECT 1 FROM manufacturing_routes
    WHERE item_code = $1 AND is_active = TRUE
) AS has_route;

-- name: GetStandardRouteForItem :one
-- Picks the route effective on the reference date (defaults to today when $3 is NULL):
-- APROVADA/active, within its validity window; prefers standard, then the most
-- recently-effective revision.
SELECT * FROM manufacturing_routes
WHERE item_code = $1
  AND COALESCE(mask, '') = COALESCE($2, '')
  AND is_active = TRUE
  AND (valid_from IS NULL OR valid_from <= COALESCE($3::DATE, CURRENT_DATE))
  AND (valid_to   IS NULL OR valid_to   >= COALESCE($3::DATE, CURRENT_DATE))
ORDER BY is_standard DESC, valid_from DESC NULLS LAST, alternative
LIMIT 1;

-- ─── route_operations ────────────────────────────────────────────────────────

-- name: AddRouteOperation :one
INSERT INTO route_operations (
    route_id, sequence, operation_id, work_center_id,
    standard_time, setup_time,
    run_time, labor_time, run_time_base_qty,
    queue_time, wait_time, move_time, crew_size, time_unit,
    supplier_id, service_item_code, cost_per_unit, lead_time_days, third_party_remittance,
    situation, notes, is_active
) VALUES (
    $1, $2, $3, $4,
    $5, $6,
    $7, $8, $9,
    $10, $11, $12, $13, $14,
    $15, $16, $17, $18, $19,
    $20, $21, TRUE
) RETURNING *;

-- name: UpdateRouteOperation :one
UPDATE route_operations SET
    work_center_id = $2,
    standard_time = $3,
    setup_time = $4,
    run_time = $5,
    labor_time = $6,
    run_time_base_qty = $7,
    queue_time = $8,
    wait_time = $9,
    move_time = $10,
    crew_size = $11,
    time_unit = $12,
    supplier_id = $13,
    service_item_code = $14,
    cost_per_unit = $15,
    lead_time_days = $16,
    third_party_remittance = $17,
    situation = $18,
    notes = $19,
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
    op.run_time AS op_run_time,
    op.labor_time AS op_labor_time,
    op.run_time_base_qty AS op_run_time_base_qty,
    op.queue_time AS op_queue_time,
    op.wait_time AS op_wait_time,
    op.move_time AS op_move_time,
    op.crew_size AS op_crew_size,
    op.time_unit AS op_time_unit,
    COALESCE(ro.work_center_id, op.default_work_center_id) AS effective_work_center_id,
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
    COALESCE(ro.standard_time, op.standard_time) AS effective_hours,
    COALESCE((SELECT supplier.code FROM suppliers supplier
              WHERE supplier.id = COALESCE(ro.supplier_id, op.supplier_id)
                 OR supplier.code = COALESCE(ro.supplier_id, op.supplier_id)
              ORDER BY (supplier.id = COALESCE(ro.supplier_id, op.supplier_id)) DESC
              LIMIT 1), COALESCE(ro.supplier_id, op.supplier_id), 0)::bigint AS supplier_id,
    COALESCE(ro.service_item_code, op.service_item_code) AS service_item_code,
    COALESCE(ro.cost_per_unit, op.cost_per_unit, 0) AS cost_per_unit,
    COALESCE(ro.lead_time_days, op.lead_time_days, 0) AS lead_time_days,
    COALESCE(ro.third_party_remittance, op.third_party_remittance, 'DEMAND_ITEMS') AS remittance_type
FROM manufacturing_routes mr
JOIN route_operations ro ON ro.route_id = mr.id
JOIN operations op ON op.id = ro.operation_id
WHERE mr.item_code = $1
  AND mr.is_active = TRUE
  AND mr.is_standard = TRUE
  AND ro.is_active = TRUE
  AND op.origin IN ('EXTERNA', 'TERCEIROS');

-- ─── route_operation_resources (alternative work centers) ─────────────────────

-- name: AddRouteOpResource :one
INSERT INTO route_operation_resources (
    route_operation_id, work_center_id, priority, time_factor, is_primary
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateRouteOpResource :one
UPDATE route_operation_resources SET
    priority    = $2,
    time_factor = $3,
    updated_at  = NOW()
WHERE id = $1
RETURNING *;

-- name: GetRouteOpResource :one
SELECT * FROM route_operation_resources WHERE id = $1;

-- name: RemoveRouteOpResource :exec
DELETE FROM route_operation_resources WHERE id = $1;

-- name: ClearPrimaryResources :exec
UPDATE route_operation_resources SET is_primary = FALSE, updated_at = NOW()
WHERE route_operation_id = $1;

-- name: SetResourcePrimary :one
UPDATE route_operation_resources SET is_primary = TRUE, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SetRouteOpWorkCenter :exec
UPDATE route_operations SET work_center_id = $2, updated_at = NOW() WHERE id = $1;

-- name: ListResourcesByRouteOp :many
SELECT r.*, mt.name AS work_center_name
FROM route_operation_resources r
LEFT JOIN machine_types mt ON mt.id = r.work_center_id
WHERE r.route_operation_id = $1
ORDER BY r.is_primary DESC, r.priority, r.id;

-- name: ListResourcesByRoute :many
SELECT r.*, mt.name AS work_center_name
FROM route_operation_resources r
JOIN route_operations ro ON ro.id = r.route_operation_id
LEFT JOIN machine_types mt ON mt.id = r.work_center_id
WHERE ro.route_id = $1
ORDER BY r.route_operation_id, r.is_primary DESC, r.priority, r.id;
