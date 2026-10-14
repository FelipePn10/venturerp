-- ─── Purchase Requisitions ────────────────────────────────────────────────────

-- name: CreatePurchaseRequisition :one
INSERT INTO purchase_requisitions (code, enterprise_code, request_type_code, requester_employee_code, emission_date, notes, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetPurchaseRequisitionByCode :one
SELECT * FROM purchase_requisitions
WHERE code = $1 AND enterprise_code = sqlc.arg(enterprise_code);

-- name: ListPurchaseRequisitions :many
SELECT * FROM purchase_requisitions
WHERE is_active = TRUE AND enterprise_code = sqlc.arg(enterprise_code)
  AND ($1::BOOLEAN = FALSE OR status IN ('OPEN','PARTIAL'))
ORDER BY code DESC;

-- name: NextPurchaseRequisitionCode :one
-- Global, como a cotação: purchase_requisitions.code é único no banco inteiro.
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM purchase_requisitions;

-- name: UpdatePurchaseRequisitionStatus :execrows
UPDATE purchase_requisitions SET status = $2, updated_at = NOW()
WHERE code = $1 AND enterprise_code = sqlc.arg(enterprise_code);

-- ─── Items ────────────────────────────────────────────────────────────────────

-- name: CreatePurchaseRequisitionItem :one
INSERT INTO purchase_requisition_items (
    requisition_code, sequence, item_code, quantity, uom, cost_center_code,
    accounting_account, suggested_price, delivery_date, application, utilization_type
)
-- Itens seguem a empresa da requisição (enterprise_code).
SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
 WHERE EXISTS (SELECT 1 FROM purchase_requisitions r
                WHERE r.code = $1 AND r.enterprise_code = sqlc.arg(enterprise_code))
RETURNING *;

-- name: ListPurchaseRequisitionItems :many
SELECT i.* FROM purchase_requisition_items i
JOIN purchase_requisitions r ON r.code = i.requisition_code
WHERE i.requisition_code = $1 AND i.is_active = TRUE AND r.enterprise_code = sqlc.arg(enterprise_code)
ORDER BY i.sequence;

-- name: GetPurchaseRequisitionItem :one
SELECT i.* FROM purchase_requisition_items i
JOIN purchase_requisitions r ON r.code = i.requisition_code
WHERE i.id = $1 AND r.enterprise_code = sqlc.arg(enterprise_code);

-- name: RegisterRequisitionItemAttendance :one
UPDATE purchase_requisition_items
SET attended_qty = attended_qty + $2,
    status = CASE
        WHEN attended_qty + $2 + cancelled_qty >= quantity THEN 'ATTENDED'
        WHEN attended_qty + $2 > 0 THEN 'PARTIAL'
        ELSE status
    END
WHERE purchase_requisition_items.id = $1
  AND EXISTS (SELECT 1 FROM purchase_requisitions r
               WHERE r.code = purchase_requisition_items.requisition_code
                 AND r.enterprise_code = sqlc.arg(enterprise_code))
RETURNING *;
