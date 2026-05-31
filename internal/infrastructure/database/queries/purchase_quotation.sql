-- ─── Purchase Quotations ──────────────────────────────────────────────────────

-- name: CreatePurchaseQuotation :one
INSERT INTO purchase_quotations (code, enterprise_code, notes, created_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPurchaseQuotationByCode :one
SELECT * FROM purchase_quotations WHERE code = $1;

-- name: ListPurchaseQuotations :many
SELECT * FROM purchase_quotations
WHERE is_active = TRUE AND ($1::BOOLEAN = FALSE OR status IN ('OPEN','QUOTED'))
ORDER BY code DESC;

-- name: NextPurchaseQuotationCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM purchase_quotations;

-- name: UpdatePurchaseQuotationStatus :exec
UPDATE purchase_quotations SET status = $2, updated_at = NOW() WHERE code = $1;

-- ─── Items ────────────────────────────────────────────────────────────────────

-- name: CreatePurchaseQuotationItem :one
INSERT INTO purchase_quotation_items (
    quotation_code, sequence, item_code, quantity, uom, delivery_date,
    source_type, source_code, source_item_id, is_configured
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListPurchaseQuotationItems :many
SELECT * FROM purchase_quotation_items WHERE quotation_code = $1 ORDER BY sequence;

-- name: GetPurchaseQuotationItem :one
SELECT * FROM purchase_quotation_items WHERE id = $1;

-- ─── Suppliers ────────────────────────────────────────────────────────────────

-- name: CreatePurchaseQuotationSupplier :one
INSERT INTO purchase_quotation_suppliers (quotation_code, supplier_code)
VALUES ($1, $2)
ON CONFLICT (quotation_code, supplier_code) DO UPDATE SET invited_at = NOW()
RETURNING *;

-- name: ListPurchaseQuotationSuppliers :many
SELECT * FROM purchase_quotation_suppliers WHERE quotation_code = $1 ORDER BY supplier_code;

-- ─── Prices ───────────────────────────────────────────────────────────────────

-- name: UpsertPurchaseQuotationPrice :one
INSERT INTO purchase_quotation_prices (quotation_item_id, supplier_code, unit_price, lead_time_days, payment_term_code, notes)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (quotation_item_id, supplier_code) DO UPDATE SET
    unit_price = EXCLUDED.unit_price,
    lead_time_days = EXCLUDED.lead_time_days,
    payment_term_code = EXCLUDED.payment_term_code,
    notes = EXCLUDED.notes
RETURNING *;

-- name: ListPurchaseQuotationPricesByItem :many
SELECT * FROM purchase_quotation_prices WHERE quotation_item_id = $1 ORDER BY unit_price;

-- name: GetPurchaseQuotationPrice :one
SELECT * FROM purchase_quotation_prices WHERE id = $1;

-- name: ClearSelectionForItem :exec
UPDATE purchase_quotation_prices SET is_selected = FALSE WHERE quotation_item_id = $1;

-- name: SetPriceSelected :one
UPDATE purchase_quotation_prices SET is_selected = TRUE WHERE id = $1 RETURNING *;

-- name: ListSelectedQuotationPrices :many
SELECT p.* FROM purchase_quotation_prices p
JOIN purchase_quotation_items i ON i.id = p.quotation_item_id
WHERE i.quotation_code = $1 AND p.is_selected = TRUE
ORDER BY p.supplier_code, p.quotation_item_id;
