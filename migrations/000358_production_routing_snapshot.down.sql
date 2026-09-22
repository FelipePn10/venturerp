BEGIN;
DROP TABLE IF EXISTS production_order_operation_dependencies;
ALTER TABLE production_orders DROP COLUMN IF EXISTS routing_snapshot_at;
COMMIT;
