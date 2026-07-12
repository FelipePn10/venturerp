DROP INDEX IF EXISTS uq_mrp_suggestion_order_number_tenant;
ALTER TABLE mrp_planned_suggestions DROP COLUMN IF EXISTS order_number;
