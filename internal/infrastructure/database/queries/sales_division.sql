-- name: CreateSalesDivision :one
INSERT INTO sales_divisions (
    code, description, commercial_analysis, financial_analysis,
    is_technical_assistance, consider_delivery_promise, consider_mrp,
    allow_outside_limits, allow_free_payment_terms, minimum_delivery_days, financial_delay_days,
    pis_percentage, cofins_percentage, parent_division_id, created_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING *;

-- name: UpdateSalesDivision :one
UPDATE sales_divisions
SET description              = $2,
    commercial_analysis      = $3,
    financial_analysis       = $4,
    is_technical_assistance  = $5,
    consider_delivery_promise = $6,
    consider_mrp             = $7,
    allow_outside_limits     = $8,
    allow_free_payment_terms = $9,
    minimum_delivery_days    = $10,
    financial_delay_days     = $11,
    pis_percentage           = $12,
    cofins_percentage        = $13,
    parent_division_id       = $14,
    updated_at               = NOW()
WHERE code = $1
RETURNING *;

-- name: GetSalesDivisionByCode :one
SELECT * FROM sales_divisions WHERE code = $1;

-- name: ListSalesDivisions :many
SELECT * FROM sales_divisions ORDER BY code;

-- name: ListActiveSalesDivisions :many
SELECT * FROM sales_divisions WHERE is_active = TRUE ORDER BY code;

-- name: DeleteSalesDivision :exec
UPDATE sales_divisions SET is_active = FALSE, updated_at = NOW() WHERE code = $1;
