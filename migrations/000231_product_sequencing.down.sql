BEGIN;
DROP TRIGGER IF EXISTS trg_production_operation_enterprise ON production_order_operations;
DROP FUNCTION IF EXISTS set_production_operation_enterprise();
DROP TRIGGER IF EXISTS trg_production_sequence_enterprise ON production_sequences;
DROP FUNCTION IF EXISTS set_production_sequence_enterprise();
DROP TABLE IF EXISTS manufacturing_sequencing_settings;
ALTER TABLE production_sequences DROP COLUMN IF EXISTS machine_id, DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE production_order_operations DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE machines DROP COLUMN IF EXISTS is_critical, DROP COLUMN IF EXISTS location,
    DROP COLUMN IF EXISTS calendar_id, DROP COLUMN IF EXISTS resource_group_id, DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE machine_types DROP CONSTRAINT IF EXISTS machine_types_distinct_cost_centers;
ALTER TABLE machine_types DROP COLUMN IF EXISTS capacity_hours, DROP COLUMN IF EXISTS labor_cost_center_id,
    DROP COLUMN IF EXISTS machine_cost_center_id, DROP COLUMN IF EXISTS enterprise_id;
DROP TABLE IF EXISTS machine_calendar_intervals;
DROP TABLE IF EXISTS machine_calendars;
DROP TABLE IF EXISTS production_resource_groups;
COMMIT;
