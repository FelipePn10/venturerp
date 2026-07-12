ALTER TABLE mrp_planned_suggestions
    ADD COLUMN IF NOT EXISTS order_number BIGINT;

CREATE UNIQUE INDEX IF NOT EXISTS uq_mrp_suggestion_order_number_tenant
    ON mrp_planned_suggestions (enterprise_id, order_number)
    WHERE enterprise_id IS NOT NULL AND order_number IS NOT NULL;
