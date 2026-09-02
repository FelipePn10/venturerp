-- name: CreateSalesForecast :one
INSERT INTO sales_forecasts (item_code, mask, week, year, quantity, created_by, enterprise_id)
VALUES ($1, $2, $3, $4, $5, $6, sqlc.arg(enterprise_id))
RETURNING *;

-- name: UpdateSalesForecast :one
UPDATE sales_forecasts
SET quantity = $2, updated_at = NOW()
WHERE id = $1 AND enterprise_id = sqlc.arg(enterprise_id)
RETURNING *;

-- name: GetSalesForecastsByItem :many
SELECT * FROM sales_forecasts WHERE item_code = $1 AND enterprise_id = sqlc.arg(enterprise_id) ORDER BY year, week;

-- name: ListSalesForecastsByYear :many
SELECT * FROM sales_forecasts WHERE year = $1 AND enterprise_id = sqlc.arg(enterprise_id) ORDER BY item_code, week;

-- name: DeleteSalesForecast :exec
DELETE FROM sales_forecasts WHERE id = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: ListForecastSalesOrderHistory :many
SELECT
    soi.item_code,
    COALESCE(NULLIF(soi.mask, ''), '')::text AS mask,
    date_trunc('month', so.emission_date)::date AS period_month,
    COALESCE(SUM(soi.requested_qty - soi.cancelled_qty), 0)::numeric AS quantity
FROM public.sales_order_items soi
JOIN public.sales_orders so ON so.code = soi.sales_order_code
WHERE so.is_active = TRUE
  AND so.enterprise_code = sqlc.arg(enterprise_code)
  AND soi.is_active = TRUE
  AND so.emission_date BETWEEN $1 AND $2
  AND so.status <> 'CANCELLED'
  AND so.is_blocked = FALSE
  AND so.release_status IN ('RELEASED', 'MANUAL_RELEASED')
  AND so.commercial_analysis_status <> 'REJECTED'
  AND so.financial_analysis_status <> 'REJECTED'
  AND soi.status <> 'CANCELLED'
  AND (cardinality($3::bigint[]) = 0 OR soi.item_code = ANY($3::bigint[]))
GROUP BY soi.item_code, COALESCE(NULLIF(soi.mask, ''), ''), date_trunc('month', so.emission_date)::date
ORDER BY soi.item_code, period_month;

-- name: ListForecastFiscalHistory :many
SELECT
    fei.item_code,
    NULL::text AS mask,
    date_trunc('month', fe.data_emissao)::date AS period_month,
    COALESCE(SUM(fei.quantity), 0)::numeric AS quantity
FROM public.fiscal_exit_items fei
JOIN public.fiscal_exits fe ON fe.id = fei.fiscal_exit_id
WHERE fe.is_active = TRUE
  AND fe.enterprise_id = sqlc.arg(enterprise_id)
  AND fe.status = 'AUTHORIZED'
  AND fe.data_emissao BETWEEN $1 AND $2
  AND fei.item_code IS NOT NULL
  AND (cardinality($3::bigint[]) = 0 OR fei.item_code = ANY($3::bigint[]))
GROUP BY fei.item_code, date_trunc('month', fe.data_emissao)::date
ORDER BY fei.item_code, period_month;

-- name: CreateSalesForecastBlock :one
INSERT INTO sales_forecast_blocks (start_date, end_date, reason, created_by, enterprise_id)
VALUES ($1, $2, $3, $4, sqlc.arg(enterprise_id))
RETURNING *;

-- name: ListSalesForecastBlocks :many
SELECT * FROM sales_forecast_blocks WHERE enterprise_id=sqlc.arg(enterprise_id) ORDER BY start_date;

-- name: IsForecastBlocked :one
SELECT EXISTS(
    SELECT 1 FROM sales_forecast_blocks
    WHERE start_date <= $1 AND end_date >= $1 AND enterprise_id=sqlc.arg(enterprise_id)
) AS blocked;

-- name: DeleteSalesForecastBlock :exec
DELETE FROM sales_forecast_blocks WHERE id = $1 AND enterprise_id=sqlc.arg(enterprise_id);

-- name: CreateAppropriationTable :one
INSERT INTO appropriation_tables (
    description, monday_pct, tuesday_pct, wednesday_pct,
    thursday_pct, friday_pct, saturday_pct, sunday_pct,
    is_default, created_by, enterprise_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, sqlc.arg(enterprise_id))
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
WHERE id = $1 AND enterprise_id=sqlc.arg(enterprise_id)
RETURNING *;

-- name: GetDefaultAppropriationTable :one
SELECT * FROM appropriation_tables WHERE is_default = TRUE AND enterprise_id=sqlc.arg(enterprise_id) LIMIT 1;

-- name: ListAppropriationTables :many
SELECT * FROM appropriation_tables WHERE enterprise_id=sqlc.arg(enterprise_id) ORDER BY id;

-- name: ClearDefaultAppropriationTable :exec
UPDATE appropriation_tables SET is_default = FALSE WHERE enterprise_id=sqlc.arg(enterprise_id);

-- name: SetSingleDefaultAppropriationTable :exec
UPDATE appropriation_tables SET is_default = TRUE WHERE id = $1 AND enterprise_id=sqlc.arg(enterprise_id);
