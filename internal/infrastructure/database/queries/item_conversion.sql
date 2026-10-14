-- Registros do item são da empresa do item (items.enterprise_id).

-- ─── Item Unit Conversions ────────────────────────────────────────────────────

-- name: CreateItemUnitConversion :one
INSERT INTO item_unit_conversions (item_code, mask, from_uom, to_uom, factor, rounding_percent, tolerance_value, tolerance_type, created_by)
SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9
 WHERE EXISTS (SELECT 1 FROM items i WHERE i.code = $1 AND i.enterprise_id = $10)
ON CONFLICT (item_code, mask, from_uom, to_uom)
DO UPDATE SET factor = EXCLUDED.factor, rounding_percent=EXCLUDED.rounding_percent, tolerance_value=EXCLUDED.tolerance_value, tolerance_type=EXCLUDED.tolerance_type, is_active = TRUE
RETURNING *;

-- name: ListItemUnitConversions :many
SELECT * FROM item_unit_conversions
WHERE item_code = $1 AND is_active = TRUE
  AND EXISTS (SELECT 1 FROM items i WHERE i.code = item_unit_conversions.item_code AND i.enterprise_id = $2)
ORDER BY mask, from_uom, to_uom;

-- name: GetItemUnitConversion :one
SELECT * FROM item_unit_conversions
WHERE item_code = $1 AND mask = $2 AND from_uom = $3 AND to_uom = $4 AND is_active = TRUE
  AND EXISTS (SELECT 1 FROM items i WHERE i.code = item_unit_conversions.item_code AND i.enterprise_id = $5);

-- name: DeleteItemUnitConversion :execrows
UPDATE item_unit_conversions SET is_active = FALSE
WHERE item_unit_conversions.id = $1
  AND EXISTS (SELECT 1 FROM items i WHERE i.code = item_unit_conversions.item_code AND i.enterprise_id = $2);
