-- ─── Purchase Requisitions ────────────────────────────────────────────────────

-- name: CreatePurchaseRequisition :one
INSERT INTO purchase_requisitions (code, enterprise_code, request_type_code, requester_employee_code, emission_date, notes, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetPurchaseRequisitionByCode :one
SELECT * FROM purchase_requisitions WHERE code = $1;

-- name: ListPurchaseRequisitions :many
SELECT * FROM purchase_requisitions
WHERE is_active = TRUE AND ($1::BOOLEAN = FALSE OR status IN ('OPEN','PARTIAL'))
ORDER BY code DESC;

-- name: NextPurchaseRequisitionCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM purchase_requisitions;

-- name: UpdatePurchaseRequisitionStatus :exec
UPDATE purchase_requisitions SET status = $2, updated_at = NOW() WHERE code = $1;

-- ─── Items ────────────────────────────────────────────────────────────────────

-- name: CreatePurchaseRequisitionItem :one
INSERT INTO purchase_requisition_items (
    requisition_code, sequence, item_code, quantity, uom, cost_center_code,
    accounting_account, suggested_price, delivery_date, application, utilization_type
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: ListPurchaseRequisitionItems :many
SELECT * FROM purchase_requisition_items
WHERE requisition_code = $1 AND is_active = TRUE
ORDER BY sequence;

-- name: GetPurchaseRequisitionItem :one
SELECT * FROM purchase_requisition_items WHERE id = $1;

-- name: RegisterRequisitionItemAttendance :one
UPDATE purchase_requisition_items
SET attended_qty = attended_qty + $2,
    status = CASE
        WHEN attended_qty + $2 + cancelled_qty >= quantity THEN 'ATTENDED'
        WHEN attended_qty + $2 > 0 THEN 'PARTIAL'
        ELSE status
    END
WHERE id = $1
RETURNING *;
