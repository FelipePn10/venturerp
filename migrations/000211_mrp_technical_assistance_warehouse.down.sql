DROP INDEX IF EXISTS idx_mrp_suggestions_oat_warehouse;
ALTER TABLE production_orders DROP COLUMN IF EXISTS warehouse_id;
ALTER TABLE planned_orders DROP COLUMN IF EXISTS warehouse_code;
ALTER TABLE mrp_planned_suggestions DROP COLUMN IF EXISTS warehouse_code;
