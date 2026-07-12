-- name: ListOpenSalesOrderDemands :many
SELECT
    soi.item_code,
    soi.mask,
    GREATEST(soi.requested_qty - soi.attended_qty - soi.cancelled_qty, 0)::numeric AS quantity,
    COALESCE(soi.delivery_date, so.delivery_date, CURRENT_DATE)::date AS need_date,
    soi.code AS source_code,
    soi.warehouse_code,
    COALESCE(sd.is_technical_assistance, FALSE)::boolean AS technical_assistance,
    (e.id <> @enterprise_id)::boolean AS inter_factory,
    COALESCE(CASE WHEN e.id <> @enterprise_id THEN e.code END, 0)::bigint AS source_enterprise_code,
    CASE WHEN e.id <> @enterprise_id THEN COALESCE(pif.auto_release, FALSE) ELSE FALSE END::boolean AS auto_release
FROM sales_order_items soi
JOIN sales_orders so ON so.code = soi.sales_order_code
JOIN enterprise e ON e.code = so.enterprise_code
LEFT JOIN sales_divisions sd ON sd.id = so.sales_division_code AND sd.is_active = TRUE
LEFT JOIN production_plan_inter_factories pif
  ON pif.plan_code = @plan_code
 AND pif.enterprise_id = @enterprise_id
 AND pif.source_enterprise_id = e.id
WHERE (e.id = @enterprise_id OR (so.origin = 'INTER_FACTORY' AND pif.source_enterprise_id IS NOT NULL))
  AND so.is_active = TRUE
  AND so.is_blocked = FALSE
  AND so.status NOT IN ('F', 'CANCELLED')
  AND soi.is_active = TRUE
  AND soi.status IN ('OPEN', 'PARTIAL')
  AND soi.requested_qty > soi.attended_qty + soi.cancelled_qty
  AND (sqlc.narg(sales_order_item_code)::bigint IS NULL OR soi.code = sqlc.narg(sales_order_item_code))
ORDER BY need_date, soi.code;

-- name: ResolveClassificationItemCodes :many
WITH RECURSIVE selected AS (
    SELECT classification.id
    FROM item_classifications classification
    JOIN item_classification_masks mask ON mask.id = classification.mask_id
    WHERE mask.code::text = sqlc.arg(classification)::text
      AND classification.code = ANY(@class_codes::text[])
      AND classification.is_active = TRUE
    UNION ALL
    SELECT child.id
    FROM item_classifications child
    JOIN selected parent ON child.parent_id = parent.id
    WHERE child.is_active = TRUE
)
SELECT DISTINCT assignment.item_code
FROM item_classification_assignments assignment
JOIN selected ON selected.id = assignment.classification_id
WHERE assignment.enterprise_id = @enterprise_id
ORDER BY assignment.item_code;
