ALTER TABLE planned_orders
    ADD COLUMN IF NOT EXISTS mrp_suggestion_code BIGINT;

CREATE UNIQUE INDEX IF NOT EXISTS uq_planned_orders_mrp_suggestion_tenant
    ON planned_orders (enterprise_id, mrp_suggestion_code)
    WHERE mrp_suggestion_code IS NOT NULL;
