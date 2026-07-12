DROP INDEX IF EXISTS uq_planned_orders_mrp_suggestion_tenant;

ALTER TABLE planned_orders
    DROP COLUMN IF EXISTS mrp_suggestion_code;
