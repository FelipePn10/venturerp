-- name: CreateProductionPlan :one
INSERT INTO production_plans (
    code, name, independent_demands, group_same_date_orders,
    planning_types, classification, class_item_codes, order_item_code,
    parameters, created_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateProductionPlan :one
UPDATE production_plans
SET name                  = $2,
    independent_demands   = $3,
    group_same_date_orders = $4,
    planning_types        = $5,
    classification        = $6,
    class_item_codes      = $7,
    order_item_code       = $8,
    parameters            = $9,
    updated_at            = NOW()
WHERE code = $1
RETURNING *;

-- name: GetProductionPlanByCode :one
SELECT * FROM production_plans WHERE code = $1 AND is_active = TRUE;

-- name: ListProductionPlans :many
SELECT * FROM production_plans WHERE is_active = TRUE ORDER BY code;

-- name: DeleteProductionPlan :exec
UPDATE production_plans SET is_active = FALSE, updated_at = NOW() WHERE code = $1;

-- name: UpdateProductionPlanLastCalculated :exec
UPDATE production_plans SET last_calculated_at = NOW(), updated_at = NOW() WHERE code = $1;
