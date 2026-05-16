-- name: NextSalesOrderNumber :one
INSERT INTO public.sales_order_sequences (enterprise_code, last_number)
VALUES ($1, 1)
ON CONFLICT (enterprise_code)
DO UPDATE SET last_number = sales_order_sequences.last_number + 1
RETURNING last_number;

-- name: CreateSalesOrder :one
INSERT INTO public.sales_orders (
    order_number, enterprise_code, status, origin,
    emission_date, delivery_date, delivery_date_firm, digit_date,
    customer_code, billing_address_code, shipping_address_code,
    representative_code, plan_code, sales_division_code,
    commission_pct, tax_type_code, presence_indicator, sales_channel,
    default_nf_type, price_table_code, currency_code,
    payment_term_code, additional_days, bearer_code, sale_date,
    total_weight_net, total_weight_gross,
    total_gross, total_net, total_net_no_st, total_with_ipi_with_st,
    notes, obs_customer, is_blocked, block_reason, is_firm, created_by
)
VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8,
    $9, $10, $11,
    $12, $13, $14,
    $15, $16, $17, $18,
    $19, $20, $21,
    $22, $23, $24, $25,
    $26, $27,
    $28, $29, $30, $31,
    $32, $33, $34, $35, $36, $37
)
RETURNING *;

-- name: UpdateSalesOrder :one
UPDATE public.sales_orders
SET
    status               = $1,
    origin               = $2,
    delivery_date        = $3,
    delivery_date_firm   = $4,
    customer_code        = $5,
    billing_address_code = $6,
    shipping_address_code= $7,
    representative_code  = $8,
    plan_code            = $9,
    sales_division_code  = $10,
    commission_pct       = $11,
    tax_type_code        = $12,
    presence_indicator   = $13,
    sales_channel        = $14,
    default_nf_type      = $15,
    price_table_code     = $16,
    currency_code        = $17,
    payment_term_code    = $18,
    additional_days      = $19,
    bearer_code          = $20,
    sale_date            = $21,
    total_weight_net     = $22,
    total_weight_gross   = $23,
    total_gross          = $24,
    total_net            = $25,
    total_net_no_st      = $26,
    total_with_ipi_with_st = $27,
    notes                = $28,
    obs_customer         = $29,
    is_firm              = $30,
    updated_at           = NOW()
WHERE code = $31 AND is_active = TRUE
RETURNING *;

-- name: GetSalesOrderByCode :one
SELECT * FROM public.sales_orders WHERE code = $1;

-- name: ListSalesOrders :many
SELECT * FROM public.sales_orders WHERE is_active = TRUE ORDER BY emission_date DESC, order_number DESC;

-- name: ListSalesOrdersByCustomer :many
SELECT * FROM public.sales_orders WHERE customer_code = $1 AND is_active = TRUE ORDER BY emission_date DESC;

-- name: ListSalesOrdersByStatus :many
SELECT * FROM public.sales_orders WHERE status = $1 AND is_active = TRUE ORDER BY emission_date DESC;

-- name: ListSalesOrdersByDateRange :many
SELECT * FROM public.sales_orders
WHERE emission_date BETWEEN $1 AND $2 AND is_active = TRUE
ORDER BY emission_date DESC;

-- name: CancelSalesOrder :exec
UPDATE public.sales_orders
SET status = 'CANCELLED', is_active = FALSE, updated_at = NOW()
WHERE code = $1;

-- name: BlockSalesOrder :exec
UPDATE public.sales_orders
SET is_blocked = TRUE, block_reason = $2, updated_at = NOW()
WHERE code = $1;

-- name: UnblockSalesOrder :exec
UPDATE public.sales_orders
SET is_blocked = FALSE, block_reason = NULL, updated_at = NOW()
WHERE code = $1;

-- name: ChangeSalesOrderStatus :exec
UPDATE public.sales_orders
SET status = $2, updated_at = NOW()
WHERE code = $1;

-- =========================================================
-- SALES ORDER ITEMS
-- =========================================================

-- name: CreateSalesOrderItem :one
INSERT INTO public.sales_order_items (
    sales_order_code, sequence, item_code, mask, digit_date,
    nf_type, sales_uom, warehouse_code, price_table_code,
    requested_qty, unit_price, attended_qty, cancelled_qty,
    delivery_date, delivery_date_firm, customer_delivery, lot, coupon_delivery, paid_at_cashier,
    ipi_pct, icms_pct, pis_pct, cofins_pct, st_pct, discount_pct,
    total_gross, total_net, total_net_with_ipi, total_ipi, total_st,
    unit_weight_net, unit_weight_gross, status, notes
)
VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12, $13,
    $14, $15, $16, $17, $18, $19,
    $20, $21, $22, $23, $24, $25,
    $26, $27, $28, $29, $30,
    $31, $32, $33, $34
)
RETURNING *;

-- name: UpdateSalesOrderItem :one
UPDATE public.sales_order_items
SET
    requested_qty       = $1,
    unit_price          = $2,
    attended_qty        = $3,
    cancelled_qty       = $4,
    delivery_date       = $5,
    delivery_date_firm  = $6,
    customer_delivery   = $7,
    lot                 = $8,
    coupon_delivery     = $9,
    paid_at_cashier     = $10,
    ipi_pct             = $11,
    icms_pct            = $12,
    pis_pct             = $13,
    cofins_pct          = $14,
    st_pct              = $15,
    discount_pct        = $16,
    total_gross         = $17,
    total_net           = $18,
    total_net_with_ipi  = $19,
    total_ipi           = $20,
    total_st            = $21,
    unit_weight_net     = $22,
    unit_weight_gross   = $23,
    status              = $24,
    notes               = $25,
    updated_at          = NOW()
WHERE code = $26 AND is_active = TRUE
RETURNING *;

-- name: ListSalesOrderItems :many
SELECT * FROM public.sales_order_items
WHERE sales_order_code = $1 AND is_active = TRUE
ORDER BY sequence;

-- name: CancelSalesOrderItem :exec
UPDATE public.sales_order_items
SET status = 'CANCELLED', cancelled_qty = requested_qty, is_active = FALSE, updated_at = NOW()
WHERE code = $1;
