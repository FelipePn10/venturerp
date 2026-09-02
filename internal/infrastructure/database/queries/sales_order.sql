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
    notes, obs_customer, is_blocked, block_reason, is_firm,
    representative_order_number, is_nfce, street, street_number, foreign_document,
    collection_establishment_code, nf_type_description, carrier_code, freight_type,
    freight_value, insurance_value, volume_quantity, volume_type, net_weight,
    gross_weight, discount_value, surcharge_value, project_code, project_name,
    created_by
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
    $32, $33, $34, $35, $36,
    $37, $38, $39, $40, $41,
    $42, $43, $44, $45,
    $46, $47, $48, $49, $50,
    $51, $52, $53, $54, $55,
    $56
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
    representative_order_number = $31,
    is_nfce              = $32,
    street               = $33,
    street_number        = $34,
    foreign_document     = $35,
    collection_establishment_code = $36,
    nf_type_description  = $37,
    carrier_code         = $38,
    freight_type         = $39,
    freight_value        = $40,
    insurance_value      = $41,
    volume_quantity      = $42,
    volume_type          = $43,
    net_weight           = $44,
    gross_weight         = $45,
    discount_value       = $46,
    surcharge_value      = $47,
    project_code         = $48,
    project_name         = $49,
    updated_at           = NOW()
WHERE code = $50 AND enterprise_code = sqlc.arg(tenant_enterprise_code) AND is_active = TRUE
RETURNING *;

-- name: GetSalesOrderByCode :one
SELECT * FROM public.sales_orders WHERE code = $1 AND enterprise_code = sqlc.arg(tenant_enterprise_code);

-- name: ListSalesOrders :many
SELECT * FROM public.sales_orders WHERE enterprise_code = sqlc.arg(tenant_enterprise_code) AND is_active = TRUE ORDER BY emission_date DESC, order_number DESC;

-- name: ListSalesOrdersByCustomer :many
SELECT * FROM public.sales_orders WHERE customer_code = $1 AND enterprise_code = sqlc.arg(tenant_enterprise_code) AND is_active = TRUE ORDER BY emission_date DESC;

-- name: ListSalesOrdersByStatus :many
SELECT * FROM public.sales_orders WHERE status = $1 AND enterprise_code = sqlc.arg(tenant_enterprise_code) AND is_active = TRUE ORDER BY emission_date DESC;

-- name: ListSalesOrdersByDateRange :many
SELECT * FROM public.sales_orders
WHERE emission_date BETWEEN $1 AND $2 AND enterprise_code = sqlc.arg(tenant_enterprise_code) AND is_active = TRUE
ORDER BY emission_date DESC;

-- name: ListSalesOrdersAdvanced :many
SELECT * FROM public.sales_orders
WHERE enterprise_code = sqlc.arg(tenant_enterprise_code) AND is_active = TRUE
  AND (sqlc.narg('search')::text IS NULL OR EXISTS (
    SELECT 1 FROM public.customers c
    WHERE c.code = sales_orders.customer_code
      AND (c.name ILIKE '%' || sqlc.narg('search')::text || '%'
        OR c.trade_name ILIKE '%' || sqlc.narg('search')::text || '%'
        OR c.document_number ILIKE '%' || sqlc.narg('search')::text || '%')
  ))
  AND (sqlc.narg('customer_code')::bigint IS NULL OR customer_code = sqlc.narg('customer_code')::bigint)
  AND (sqlc.narg('item_code')::bigint IS NULL OR EXISTS (
    SELECT 1 FROM public.sales_order_items soi
    WHERE soi.sales_order_code = sales_orders.code
      AND soi.item_code = sqlc.narg('item_code')::bigint
  ))
  AND (sqlc.narg('representative_code')::bigint IS NULL OR representative_code = sqlc.narg('representative_code')::bigint)
  AND (sqlc.narg('payment_term_code')::bigint IS NULL OR payment_term_code = sqlc.narg('payment_term_code')::bigint)
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (sqlc.narg('commercial_analysis_status')::text IS NULL OR commercial_analysis_status = sqlc.narg('commercial_analysis_status')::text)
  AND (sqlc.narg('financial_analysis_status')::text IS NULL OR financial_analysis_status = sqlc.narg('financial_analysis_status')::text)
  AND (sqlc.narg('release_status')::text IS NULL OR release_status = sqlc.narg('release_status')::text)
  AND (sqlc.narg('conference_status')::text IS NULL OR conference_status = sqlc.narg('conference_status')::text)
  AND (sqlc.narg('is_blocked')::boolean IS NULL OR is_blocked = sqlc.narg('is_blocked')::boolean)
  AND (sqlc.narg('workflow_status')::text IS NULL OR
       (sqlc.narg('workflow_status')::text = 'ANALISADO' AND (commercial_analysis_status <> 'NOT_ANALYZED' OR financial_analysis_status <> 'NOT_ANALYZED')) OR
       (sqlc.narg('workflow_status')::text = 'ATENDIDO' AND attended_at IS NOT NULL) OR
       (sqlc.narg('workflow_status')::text = 'CONFERIDO' AND conference_status = 'CONFERRED') OR
       (sqlc.narg('workflow_status')::text = 'ATRASADO' AND delivery_date < CURRENT_DATE AND status NOT IN ('F','CANCELLED')) OR
       (sqlc.narg('workflow_status')::text = 'FATURADO' AND status = 'F'))
  AND (sqlc.narg('emission_from')::date IS NULL OR emission_date >= sqlc.narg('emission_from')::date)
  AND (sqlc.narg('emission_to')::date IS NULL OR emission_date <= sqlc.narg('emission_to')::date)
  AND (sqlc.narg('delivery_from')::date IS NULL OR delivery_date >= sqlc.narg('delivery_from')::date)
  AND (sqlc.narg('delivery_to')::date IS NULL OR delivery_date <= sqlc.narg('delivery_to')::date)
ORDER BY emission_date DESC, order_number DESC, code DESC
LIMIT sqlc.arg('page_limit') OFFSET sqlc.arg('page_offset');

-- name: SalesOrderReport :one
SELECT
  COUNT(*)::bigint AS total_orders,
  COALESCE(SUM(total_gross),0)::numeric AS total_gross,
  COALESCE(SUM(total_net),0)::numeric AS total_net,
  COUNT(*) FILTER (WHERE status IN ('R','A','P'))::bigint AS open_count,
  COUNT(*) FILTER (WHERE status='P')::bigint AS confirmed_count,
  COUNT(*) FILTER (WHERE status='F')::bigint AS invoiced_count,
  COUNT(*) FILTER (WHERE status='CANCELLED')::bigint AS cancelled_count,
  COUNT(*) FILTER (WHERE is_blocked)::bigint AS blocked_count,
  COUNT(*) FILTER (WHERE commercial_analysis_status='NOT_ANALYZED')::bigint AS commercial_pending_count,
  COUNT(*) FILTER (WHERE financial_analysis_status='NOT_ANALYZED')::bigint AS financial_pending_count,
  COUNT(*) FILTER (WHERE conference_status='PENDING')::bigint AS conference_pending_count,
  COUNT(*) FILTER (WHERE delivery_date < CURRENT_DATE AND status NOT IN ('F','CANCELLED'))::bigint AS delayed_count
FROM public.sales_orders
WHERE enterprise_code = sqlc.arg(tenant_enterprise_code) AND is_active = TRUE
  AND (sqlc.narg('search')::text IS NULL OR EXISTS (
    SELECT 1 FROM public.customers c
    WHERE c.code = sales_orders.customer_code
      AND (c.name ILIKE '%' || sqlc.narg('search')::text || '%'
        OR c.trade_name ILIKE '%' || sqlc.narg('search')::text || '%'
        OR c.document_number ILIKE '%' || sqlc.narg('search')::text || '%')
  ))
  AND (sqlc.narg('customer_code')::bigint IS NULL OR customer_code = sqlc.narg('customer_code')::bigint)
  AND (sqlc.narg('item_code')::bigint IS NULL OR EXISTS (
    SELECT 1 FROM public.sales_order_items soi
    WHERE soi.sales_order_code = sales_orders.code
      AND soi.item_code = sqlc.narg('item_code')::bigint
  ))
  AND (sqlc.narg('representative_code')::bigint IS NULL OR representative_code = sqlc.narg('representative_code')::bigint)
  AND (sqlc.narg('payment_term_code')::bigint IS NULL OR payment_term_code = sqlc.narg('payment_term_code')::bigint)
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (sqlc.narg('commercial_analysis_status')::text IS NULL OR commercial_analysis_status = sqlc.narg('commercial_analysis_status')::text)
  AND (sqlc.narg('financial_analysis_status')::text IS NULL OR financial_analysis_status = sqlc.narg('financial_analysis_status')::text)
  AND (sqlc.narg('release_status')::text IS NULL OR release_status = sqlc.narg('release_status')::text)
  AND (sqlc.narg('conference_status')::text IS NULL OR conference_status = sqlc.narg('conference_status')::text)
  AND (sqlc.narg('is_blocked')::boolean IS NULL OR is_blocked = sqlc.narg('is_blocked')::boolean)
  AND (sqlc.narg('workflow_status')::text IS NULL OR
       (sqlc.narg('workflow_status')::text = 'ANALISADO' AND (commercial_analysis_status <> 'NOT_ANALYZED' OR financial_analysis_status <> 'NOT_ANALYZED')) OR
       (sqlc.narg('workflow_status')::text = 'ATENDIDO' AND attended_at IS NOT NULL) OR
       (sqlc.narg('workflow_status')::text = 'CONFERIDO' AND conference_status = 'CONFERRED') OR
       (sqlc.narg('workflow_status')::text = 'ATRASADO' AND delivery_date < CURRENT_DATE AND status NOT IN ('F','CANCELLED')) OR
       (sqlc.narg('workflow_status')::text = 'FATURADO' AND status = 'F'))
  AND (sqlc.narg('emission_from')::date IS NULL OR emission_date >= sqlc.narg('emission_from')::date)
  AND (sqlc.narg('emission_to')::date IS NULL OR emission_date <= sqlc.narg('emission_to')::date)
  AND (sqlc.narg('delivery_from')::date IS NULL OR delivery_date >= sqlc.narg('delivery_from')::date)
  AND (sqlc.narg('delivery_to')::date IS NULL OR delivery_date <= sqlc.narg('delivery_to')::date);

-- name: CancelSalesOrder :execrows
UPDATE public.sales_orders
SET status = 'CANCELLED', cancel_reason = $2, cancel_complement = $3, updated_at = NOW()
WHERE code = $1 AND enterprise_code = sqlc.arg(tenant_enterprise_code);

-- name: BlockSalesOrder :execrows
UPDATE public.sales_orders
SET is_blocked = TRUE, block_reason = $2, updated_at = NOW()
WHERE code = $1 AND enterprise_code = sqlc.arg(tenant_enterprise_code);

-- name: UnblockSalesOrder :execrows
UPDATE public.sales_orders
SET is_blocked = FALSE, block_reason = NULL, updated_at = NOW()
WHERE code = $1 AND enterprise_code = sqlc.arg(tenant_enterprise_code);

-- name: ChangeSalesOrderStatus :execrows
UPDATE public.sales_orders
SET status = $2, updated_at = NOW()
WHERE code = $1 AND enterprise_code = sqlc.arg(tenant_enterprise_code);

-- name: AnalyzeSalesOrder :execrows
UPDATE public.sales_orders
SET commercial_analysis_status = CASE WHEN $2 = 'COMMERCIAL' THEN $3 ELSE commercial_analysis_status END,
    financial_analysis_status = CASE WHEN $2 = 'FINANCIAL' THEN $3 ELSE financial_analysis_status END,
    updated_at = NOW()
WHERE code = $1 AND enterprise_code = sqlc.arg(tenant_enterprise_code);

-- name: ReleaseSalesOrder :execrows
WITH input AS (SELECT sqlc.arg(release_status)::text AS release_status)
UPDATE public.sales_orders
SET release_status = input.release_status,
    is_blocked = CASE WHEN input.release_status = 'BLOCKED' THEN TRUE ELSE FALSE END,
    block_reason = CASE WHEN input.release_status = 'BLOCKED' THEN sqlc.arg(block_reason) ELSE NULL END,
    updated_at = NOW()
FROM input
WHERE code = sqlc.arg(code) AND enterprise_code = sqlc.arg(tenant_enterprise_code);

-- name: AttendSalesOrder :execrows
UPDATE public.sales_orders
SET attended_reason = $2,
    attended_at = COALESCE($3, NOW()),
    updated_at = NOW()
WHERE code = $1 AND enterprise_code = sqlc.arg(tenant_enterprise_code);

-- name: ConferSalesOrder :execrows
UPDATE public.sales_orders
SET conference_status = $2,
    updated_at = NOW()
WHERE code = $1 AND enterprise_code = sqlc.arg(tenant_enterprise_code);

-- name: SaveSalesOrderDelayReason :execrows
UPDATE public.sales_orders
SET delay_reason = $2,
    delay_action = $3,
    updated_at = NOW()
WHERE code = $1 AND enterprise_code = sqlc.arg(tenant_enterprise_code);

-- name: InsertSalesOrderEvent :exec
INSERT INTO public.sales_order_events (
    sales_order_code, event_type, area, reason, complement, event_date, created_by
) VALUES (
    $1, $2, $3, $4, $5, COALESCE($6, NOW()), $7
);

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
SELECT
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12, $13,
    $14, $15, $16, $17, $18, $19,
    $20, $21, $22, $23, $24, $25,
    $26, $27, $28, $29, $30,
    $31, $32, $33, $34
WHERE EXISTS (
    SELECT 1 FROM public.sales_orders
    WHERE code=$1 AND enterprise_code=sqlc.arg(tenant_enterprise_code)
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
WHERE sales_order_items.code = $26 AND sales_order_items.is_active = TRUE
  AND EXISTS (
      SELECT 1 FROM public.sales_orders sales_order
      WHERE sales_order.code=sales_order_items.sales_order_code
        AND sales_order.enterprise_code=sqlc.arg(tenant_enterprise_code)
  )
RETURNING *;

-- name: GetSalesOrderItem :one
SELECT item.* FROM public.sales_order_items item
JOIN public.sales_orders sales_order ON sales_order.code=item.sales_order_code
WHERE item.code=sqlc.arg(item_code) AND item.is_active=TRUE
  AND sales_order.enterprise_code=sqlc.arg(tenant_enterprise_code);

-- name: ListSalesOrderItems :many
SELECT * FROM public.sales_order_items
WHERE sales_order_items.sales_order_code = $1 AND sales_order_items.is_active = TRUE
  AND EXISTS (
      SELECT 1 FROM public.sales_orders sales_order
      WHERE sales_order.code=sales_order_items.sales_order_code
        AND sales_order.enterprise_code=sqlc.arg(tenant_enterprise_code)
  )
ORDER BY sequence;

-- name: CancelSalesOrderItem :execrows
UPDATE public.sales_order_items
SET status = 'CANCELLED', cancelled_qty = requested_qty, is_active = FALSE, updated_at = NOW()
WHERE sales_order_items.code = $1
  AND EXISTS (
      SELECT 1 FROM public.sales_orders sales_order
      WHERE sales_order.code=sales_order_items.sales_order_code
        AND sales_order.enterprise_code=sqlc.arg(tenant_enterprise_code)
  );
