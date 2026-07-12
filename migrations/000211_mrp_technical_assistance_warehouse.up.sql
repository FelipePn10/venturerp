ALTER TABLE mrp_planned_suggestions ADD COLUMN IF NOT EXISTS warehouse_code BIGINT;
ALTER TABLE planned_orders ADD COLUMN IF NOT EXISTS warehouse_code BIGINT;
ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS warehouse_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_mrp_suggestions_oat_warehouse
    ON mrp_planned_suggestions (enterprise_id, warehouse_code)
    WHERE order_type = 'TECHNICAL_ASSISTANCE';
