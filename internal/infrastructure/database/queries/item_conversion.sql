-- ─── Item Unit Conversions ────────────────────────────────────────────────────

-- name: CreateItemUnitConversion :one
INSERT INTO item_unit_conversions (item_code, from_uom, to_uom, factor, created_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (item_code, from_uom, to_uom)
DO UPDATE SET factor = EXCLUDED.factor, is_active = TRUE
RETURNING *;

-- name: ListItemUnitConversions :many
SELECT * FROM item_unit_conversions
WHERE item_code = $1 AND is_active = TRUE
ORDER BY from_uom, to_uom;

-- name: GetItemUnitConversion :one
SELECT * FROM item_unit_conversions
WHERE item_code = $1 AND from_uom = $2 AND to_uom = $3 AND is_active = TRUE;

-- name: DeleteItemUnitConversion :exec
UPDATE item_unit_conversions SET is_active = FALSE WHERE id = $1;
