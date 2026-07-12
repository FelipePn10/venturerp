DROP INDEX IF EXISTS idx_mrp_suggestions_inter_factory;
ALTER TABLE planned_orders
    DROP COLUMN IF EXISTS auto_release,
    DROP COLUMN IF EXISTS source_enterprise_code,
    DROP COLUMN IF EXISTS inter_factory;
ALTER TABLE mrp_planned_suggestions
    DROP COLUMN IF EXISTS auto_release,
    DROP COLUMN IF EXISTS source_enterprise_code,
    DROP COLUMN IF EXISTS inter_factory;
