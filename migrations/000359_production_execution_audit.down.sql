BEGIN;
DROP TRIGGER IF EXISTS production_execution_event ON production_order_operations;
DROP FUNCTION IF EXISTS record_production_execution_event();
DROP TABLE IF EXISTS production_operation_execution_events;
DROP FUNCTION IF EXISTS protect_production_execution_event();
COMMIT;
