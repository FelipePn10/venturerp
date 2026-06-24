-- ─── cutting_plans ────────────────────────────────────────────────────────────

-- name: NextCuttingPlanCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM cutting_plans;

-- name: CreateCuttingPlan :one
INSERT INTO cutting_plans (
    code, description, cut_type, source, status,
    material_item_code, machine_code,
    kerf_mm, trim_mm, min_remnant_mm,
    warehouse_id, production_order_code, lot_consumption_mode, include_remnants,
    stock_uom, uom_factor,
    created_by
) VALUES (
    $1, $2, $3, $4, 'RASCUNHO',
    $5, $6,
    $7, $8, $9,
    $10, $11, $12, $13,
    $14, $15,
    $16
) RETURNING *;

-- name: GetCuttingPlanByID :one
SELECT * FROM cutting_plans WHERE id = $1;

-- name: ListCuttingPlans :many
SELECT * FROM cutting_plans
WHERE ($1::BOOLEAN = FALSE OR status IN ('RASCUNHO','OTIMIZADO'))
ORDER BY code DESC;

-- name: UpdateCuttingPlanResult :exec
UPDATE cutting_plans SET
    status           = $2,
    utilization_pct  = $3,
    scrap_pct        = $4,
    stock_used_count = $5,
    cut_count        = $6,
    total_demand     = $7,
    total_stock      = $8,
    updated_at       = NOW()
WHERE id = $1;

-- name: DeleteCuttingPlan :exec
DELETE FROM cutting_plans WHERE id = $1;

-- ─── cutting_plan_parts ───────────────────────────────────────────────────────

-- name: AddCuttingPlanPart :one
INSERT INTO cutting_plan_parts (
    plan_id, item_code, label, length_mm, quantity, source_ref,
    width_mm, height_mm, grain, allow_rotation, geometry,
    edge_top, edge_bottom, edge_left, edge_right, band_item_code, band_cost_per_m
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    $12, $13, $14, $15, $16, $17
) RETURNING *;

-- name: ListCuttingPlanParts :many
SELECT * FROM cutting_plan_parts WHERE plan_id = $1 ORDER BY id;

-- name: RemoveCuttingPlanPart :exec
DELETE FROM cutting_plan_parts WHERE id = $1;

-- ─── cutting_stock_pieces ─────────────────────────────────────────────────────

-- name: AddCuttingStockPiece :one
INSERT INTO cutting_stock_pieces (
    plan_id, length_mm, quantity, lot, is_remnant, remnant_id, heat_number,
    width_mm, height_mm
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9
) RETURNING *;

-- name: ListCuttingStockPieces :many
SELECT * FROM cutting_stock_pieces WHERE plan_id = $1 ORDER BY id;

-- name: RemoveCuttingStockPiece :exec
DELETE FROM cutting_stock_pieces WHERE id = $1;

-- name: DeleteRemnantStockPieces :exec
DELETE FROM cutting_stock_pieces WHERE plan_id = $1 AND remnant_id IS NOT NULL;

-- ─── cutting_patterns ─────────────────────────────────────────────────────────

-- name: DeleteCuttingPatternsByPlan :exec
DELETE FROM cutting_patterns WHERE plan_id = $1;

-- name: CreateCuttingPattern :one
INSERT INTO cutting_patterns (
    plan_id, sequence, stock_length_mm, repeat_count,
    used_mm, kerf_loss_mm, remnant_mm, utilization_pct, is_remnant,
    stock_width_mm, stock_height_mm, used_area_mm2, remnant_area_mm2,
    remnant_width_mm, remnant_height_mm
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8, $9,
    $10, $11, $12, $13,
    $14, $15
) RETURNING *;

-- name: ListCuttingPatternsByPlan :many
SELECT * FROM cutting_patterns WHERE plan_id = $1 ORDER BY sequence;

-- name: CreateCuttingPatternPlacement :one
INSERT INTO cutting_pattern_placements (
    pattern_id, sequence, part_id, label, length_mm, offset_mm,
    pos_x_mm, pos_y_mm, width_mm, height_mm, rotated, rotation_deg
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: ListCuttingPatternPlacements :many
SELECT * FROM cutting_pattern_placements WHERE pattern_id = $1 ORDER BY sequence;

-- ─── release / firmar ─────────────────────────────────────────────────────────

-- name: ReleaseCuttingPlan :exec
UPDATE cutting_plans SET
    status      = 'FIRMADO',
    released_at = NOW(),
    updated_at  = NOW()
WHERE id = $1;

-- ─── stock_remnants ───────────────────────────────────────────────────────────

-- name: CreateStockRemnant :one
INSERT INTO stock_remnants (
    item_code, warehouse_id, length_mm, lot, heat_number, certificate,
    status, unit_cost, origin_plan_id, created_by,
    width_mm, height_mm
) VALUES (
    $1, $2, $3, $4, $5, $6,
    'AVAILABLE', $7, $8, $9,
    $10, $11
) RETURNING *;

-- name: ListAvailableRemnants :many
SELECT * FROM stock_remnants
WHERE item_code = $1 AND warehouse_id = $2 AND status = 'AVAILABLE'
ORDER BY length_mm ASC, id ASC;

-- name: ListRemnantsByItem :many
SELECT * FROM stock_remnants
WHERE item_code = $1 AND ($2::BOOLEAN = FALSE OR status = 'AVAILABLE')
ORDER BY status, length_mm ASC;

-- name: GetStockRemnant :one
SELECT * FROM stock_remnants WHERE id = $1;

-- name: MarkRemnantConsumed :exec
UPDATE stock_remnants SET
    status = 'CONSUMED', consumed_plan_id = $2, updated_at = NOW()
WHERE id = $1 AND status = 'AVAILABLE';

-- ─── cutting_plan_consumptions ────────────────────────────────────────────────

-- name: AddCuttingPlanConsumption :one
INSERT INTO cutting_plan_consumptions (
    plan_id, item_code, source_type, lot, remnant_id,
    quantity, length_mm, unit_cost, total_cost, warehouse_id, movement_id
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: ListCuttingPlanConsumptions :many
SELECT * FROM cutting_plan_consumptions WHERE plan_id = $1 ORDER BY id;

-- ─── cutting_settings (singleton id=1) ────────────────────────────────────────

-- name: GetCuttingSettings :one
SELECT * FROM cutting_settings WHERE id = 1;

-- name: UpsertCuttingSettings :one
INSERT INTO cutting_settings (id, default_consumption_mode, default_min_remnant_mm, default_warehouse_id, updated_at)
VALUES (1, $1, $2, $3, NOW())
ON CONFLICT (id) DO UPDATE SET
    default_consumption_mode = EXCLUDED.default_consumption_mode,
    default_min_remnant_mm   = EXCLUDED.default_min_remnant_mm,
    default_warehouse_id     = EXCLUDED.default_warehouse_id,
    updated_at               = NOW()
RETURNING *;

-- ─── FIFO lots for automatic consumption ──────────────────────────────────────

-- name: ListAvailableLotsFIFO :many
SELECT b.lot, b.quantity, b.last_cost,
       l.heat_number, l.certificate, l.received_at
FROM stock_lot_balances b
LEFT JOIN stock_lots l ON l.item_code = b.item_code AND l.lot = b.lot
WHERE b.item_code = $1 AND b.warehouse_id = $2 AND b.quantity > 0
ORDER BY l.received_at ASC NULLS LAST, b.lot ASC;

-- ─── cutting_plan_order_costs (rateio por OP) ─────────────────────────────────

-- name: DeleteOrderCostsByPlan :exec
DELETE FROM cutting_plan_order_costs WHERE plan_id = $1;

-- name: AddCuttingPlanOrderCost :one
INSERT INTO cutting_plan_order_costs (
    plan_id, order_ref, demand_measure, allocated_cost
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: ListCuttingPlanOrderCosts :many
SELECT * FROM cutting_plan_order_costs WHERE plan_id = $1 ORDER BY order_ref;
