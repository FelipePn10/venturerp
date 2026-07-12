DROP TABLE IF EXISTS production_order_service_links;
DROP TABLE IF EXISTS production_deliveries;
DROP INDEX IF EXISTS idx_production_orders_tenant;
ALTER TABLE production_orders DROP COLUMN IF EXISTS enterprise_id;
