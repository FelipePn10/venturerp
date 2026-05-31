-- ─── Item Preferred Suppliers ─────────────────────────────────────────────────

-- name: UpsertItemPreferredSupplier :one
INSERT INTO item_preferred_suppliers (
    item_code, supplier_code, ranking, supplier_item_code, supplier_description, uom, lead_time_days, created_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (item_code, supplier_code) DO UPDATE SET
    ranking = EXCLUDED.ranking,
    supplier_item_code = EXCLUDED.supplier_item_code,
    supplier_description = EXCLUDED.supplier_description,
    uom = EXCLUDED.uom,
    lead_time_days = EXCLUDED.lead_time_days,
    is_active = TRUE
RETURNING *;

-- name: ListItemPreferredSuppliers :many
SELECT * FROM item_preferred_suppliers
WHERE item_code = $1 AND is_active = TRUE
ORDER BY ranking, supplier_code;

-- name: GetPreferredSupplierForItem :one
SELECT * FROM item_preferred_suppliers
WHERE item_code = $1 AND is_active = TRUE
ORDER BY ranking, supplier_code
LIMIT 1;

-- name: DeleteItemPreferredSupplier :exec
UPDATE item_preferred_suppliers SET is_active = FALSE WHERE id = $1;
