-- ─── Purchase Quotations ──────────────────────────────────────────────────────

-- name: CreatePurchaseQuotation :one
INSERT INTO purchase_quotations (code, enterprise_code, notes, created_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPurchaseQuotationByCode :one
SELECT * FROM purchase_quotations
WHERE code = $1 AND enterprise_code = sqlc.arg(enterprise_code);

-- name: ListPurchaseQuotations :many
SELECT * FROM purchase_quotations
WHERE is_active = TRUE AND enterprise_code = sqlc.arg(enterprise_code)
  AND ($1::BOOLEAN = FALSE OR status IN ('OPEN','QUOTED'))
ORDER BY code DESC;

-- name: NextPurchaseQuotationCode :one
-- purchase_quotations.code é único no banco inteiro (os filhos referenciam só o
-- código), então a numeração também tem de ser: por empresa, a segunda empresa
-- recebia um código já usado pela primeira e a criação falhava.
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM purchase_quotations;

-- name: UpdatePurchaseQuotationStatus :execrows
UPDATE purchase_quotations SET status = $2, updated_at = NOW()
WHERE code = $1 AND enterprise_code = sqlc.arg(enterprise_code);

-- ─── Items ────────────────────────────────────────────────────────────────────

-- name: CreatePurchaseQuotationItem :one
-- Filhos da cotação seguem a empresa dela (enterprise_code): sem cotação da
-- empresa autenticada, nada é gravado.
INSERT INTO purchase_quotation_items (
    quotation_code, sequence, item_code, quantity, uom, delivery_date,
    source_type, source_code, source_item_id, is_configured
)
SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
 WHERE EXISTS (SELECT 1 FROM purchase_quotations q
                WHERE q.code = $1 AND q.enterprise_code = sqlc.arg(enterprise_code))
RETURNING *;

-- name: ListPurchaseQuotationItems :many
SELECT i.* FROM purchase_quotation_items i
JOIN purchase_quotations q ON q.code = i.quotation_code
WHERE i.quotation_code = $1 AND q.enterprise_code = sqlc.arg(enterprise_code)
ORDER BY i.sequence;

-- name: GetPurchaseQuotationItem :one
SELECT i.* FROM purchase_quotation_items i
JOIN purchase_quotations q ON q.code = i.quotation_code
WHERE i.id = $1 AND q.enterprise_code = sqlc.arg(enterprise_code);

-- ─── Suppliers ────────────────────────────────────────────────────────────────

-- name: CreatePurchaseQuotationSupplier :one
INSERT INTO purchase_quotation_suppliers (quotation_code, supplier_code)
SELECT $1, $2
 WHERE EXISTS (SELECT 1 FROM purchase_quotations q
                WHERE q.code = $1 AND q.enterprise_code = sqlc.arg(enterprise_code))
   AND EXISTS (SELECT 1 FROM suppliers s JOIN enterprise e ON e.id = s.enterprise_id
                WHERE s.code = $2 AND e.code = sqlc.arg(enterprise_code))
ON CONFLICT (quotation_code, supplier_code) DO UPDATE SET invited_at = NOW()
RETURNING *;

-- name: ListPurchaseQuotationSuppliers :many
SELECT s.* FROM purchase_quotation_suppliers s
JOIN purchase_quotations q ON q.code = s.quotation_code
WHERE s.quotation_code = $1 AND q.enterprise_code = sqlc.arg(enterprise_code)
ORDER BY s.supplier_code;

-- ─── Prices ───────────────────────────────────────────────────────────────────

-- name: UpsertPurchaseQuotationPrice :one
INSERT INTO purchase_quotation_prices (quotation_item_id, supplier_code, unit_price, lead_time_days, payment_term_code, notes)
SELECT $1, $2, $3, $4, $5, $6
 WHERE EXISTS (SELECT 1 FROM purchase_quotation_items i
                 JOIN purchase_quotations q ON q.code = i.quotation_code
                WHERE i.id = $1 AND q.enterprise_code = sqlc.arg(enterprise_code))
   AND EXISTS (SELECT 1 FROM suppliers s JOIN enterprise e ON e.id = s.enterprise_id
                WHERE s.code = $2 AND e.code = sqlc.arg(enterprise_code))
ON CONFLICT (quotation_item_id, supplier_code) DO UPDATE SET
    unit_price = EXCLUDED.unit_price,
    lead_time_days = EXCLUDED.lead_time_days,
    payment_term_code = EXCLUDED.payment_term_code,
    notes = EXCLUDED.notes
RETURNING *;

-- name: ListPurchaseQuotationPricesByItem :many
SELECT p.* FROM purchase_quotation_prices p
JOIN purchase_quotation_items i ON i.id = p.quotation_item_id
JOIN purchase_quotations q ON q.code = i.quotation_code
WHERE p.quotation_item_id = $1 AND q.enterprise_code = sqlc.arg(enterprise_code)
ORDER BY p.unit_price;

-- name: GetPurchaseQuotationPrice :one
SELECT p.* FROM purchase_quotation_prices p
JOIN purchase_quotation_items i ON i.id = p.quotation_item_id
JOIN purchase_quotations q ON q.code = i.quotation_code
WHERE p.id = $1 AND q.enterprise_code = sqlc.arg(enterprise_code);

-- name: ClearSelectionForItem :exec
UPDATE purchase_quotation_prices SET is_selected = FALSE WHERE quotation_item_id = $1;

-- name: SetPriceSelected :one
UPDATE purchase_quotation_prices SET is_selected = TRUE WHERE id = $1 RETURNING *;

-- name: ListSelectedQuotationPrices :many
SELECT p.* FROM purchase_quotation_prices p
JOIN purchase_quotation_items i ON i.id = p.quotation_item_id
JOIN purchase_quotations q ON q.code = i.quotation_code
WHERE i.quotation_code = $1 AND p.is_selected = TRUE AND q.enterprise_code = sqlc.arg(enterprise_code)
ORDER BY p.supplier_code, p.quotation_item_id;
