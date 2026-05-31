-- ─── Purchase Price Tables ────────────────────────────────────────────────────

-- name: CreatePurchasePriceTable :one
INSERT INTO purchase_price_tables (code, description, currency_code, validity_start, validity_end, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdatePurchasePriceTable :one
UPDATE purchase_price_tables
SET description = $2, currency_code = $3, validity_start = $4, validity_end = $5,
    is_active = $6, updated_at = NOW()
WHERE code = $1
RETURNING *;

-- name: GetPurchasePriceTableByCode :one
SELECT * FROM purchase_price_tables WHERE code = $1;

-- name: ListPurchasePriceTables :many
SELECT * FROM purchase_price_tables
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextPurchasePriceTableCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM purchase_price_tables;

-- ─── Items ────────────────────────────────────────────────────────────────────

-- name: CreatePurchasePriceTableItem :one
INSERT INTO purchase_price_table_items (table_id, item_code, supplier_code, uom, price, min_qty)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (table_id, item_code, COALESCE(supplier_code, 0))
DO UPDATE SET uom = EXCLUDED.uom, price = EXCLUDED.price, min_qty = EXCLUDED.min_qty, is_active = TRUE
RETURNING *;

-- name: ListPurchasePriceTableItems :many
SELECT * FROM purchase_price_table_items
WHERE table_id = $1 AND is_active = TRUE
ORDER BY item_code;

-- name: DeletePurchasePriceTableItem :exec
UPDATE purchase_price_table_items SET is_active = FALSE WHERE id = $1;

-- name: GetPurchasePrice :one
SELECT i.* FROM purchase_price_table_items i
JOIN purchase_price_tables t ON t.id = i.table_id
WHERE t.code = $1 AND i.item_code = $2 AND i.is_active = TRUE
  AND (i.supplier_code = $3 OR i.supplier_code IS NULL)
ORDER BY (i.supplier_code = $3) DESC NULLS LAST
LIMIT 1;
