-- name: CreateSalesForecast :one
INSERT INTO sales_forecasts (item_code, mask, week, year, quantity, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateSalesForecast :one
UPDATE sales_forecasts
SET quantity = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetSalesForecastsByItem :many
SELECT * FROM sales_forecasts WHERE item_code = $1 ORDER BY year, week;

-- name: ListSalesForecastsByYear :many
SELECT * FROM sales_forecasts WHERE year = $1 ORDER BY item_code, week;

-- name: DeleteSalesForecast :exec
DELETE FROM sales_forecasts WHERE id = $1;

-- name: CreateSalesForecastBlock :one
INSERT INTO sales_forecast_blocks (start_date, end_date, reason, created_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListSalesForecastBlocks :many
SELECT * FROM sales_forecast_blocks ORDER BY start_date;

-- name: IsForecastBlocked :one
SELECT EXISTS(
    SELECT 1 FROM sales_forecast_blocks
    WHERE start_date <= $1 AND end_date >= $1
) AS blocked;

-- name: DeleteSalesForecastBlock :exec
DELETE FROM sales_forecast_blocks WHERE id = $1;

-- name: CreateAppropriationTable :one
INSERT INTO appropriation_tables (
    description, monday_pct, tuesday_pct, wednesday_pct,
    thursday_pct, friday_pct, saturday_pct, sunday_pct,
    is_default, created_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateAppropriationTable :one
UPDATE appropriation_tables
SET description   = $2,
    monday_pct    = $3,
    tuesday_pct   = $4,
    wednesday_pct = $5,
    thursday_pct  = $6,
    friday_pct    = $7,
    saturday_pct  = $8,
    sunday_pct    = $9,
    updated_at    = NOW()
WHERE id = $1
RETURNING *;

-- name: GetDefaultAppropriationTable :one
SELECT * FROM appropriation_tables WHERE is_default = TRUE LIMIT 1;

-- name: ListAppropriationTables :many
SELECT * FROM appropriation_tables ORDER BY id;

-- name: ClearDefaultAppropriationTable :exec
UPDATE appropriation_tables SET is_default = FALSE;

-- name: SetSingleDefaultAppropriationTable :exec
UPDATE appropriation_tables SET is_default = TRUE WHERE id = $1;
