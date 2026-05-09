ALTER TABLE mrp_calculation_logs
    ADD COLUMN IF NOT EXISTS code BIGSERIAL;

ALTER TABLE mrp_item_profiles
    ADD COLUMN IF NOT EXISTS code BIGSERIAL;

ALTER TABLE stock_snapshots
    RENAME COLUMN warehouse_id TO warehouse_code;

ALTER TABLE sales_order_demands
    RENAME COLUMN division_id TO division_code;

ALTER TABLE sales_order_demands
    ADD COLUMN IF NOT EXISTS code BIGSERIAL;
