BEGIN;

ALTER TABLE route_operations DROP CONSTRAINT IF EXISTS chk_route_ops_time_unit;
ALTER TABLE route_operations DROP CONSTRAINT IF EXISTS chk_route_ops_base_qty;
ALTER TABLE route_operations DROP CONSTRAINT IF EXISTS chk_route_ops_crew;
ALTER TABLE route_operations
    DROP COLUMN IF EXISTS run_time,
    DROP COLUMN IF EXISTS labor_time,
    DROP COLUMN IF EXISTS run_time_base_qty,
    DROP COLUMN IF EXISTS queue_time,
    DROP COLUMN IF EXISTS wait_time,
    DROP COLUMN IF EXISTS move_time,
    DROP COLUMN IF EXISTS crew_size,
    DROP COLUMN IF EXISTS time_unit;

ALTER TABLE operations DROP CONSTRAINT IF EXISTS chk_operations_time_unit;
ALTER TABLE operations DROP CONSTRAINT IF EXISTS chk_operations_base_qty;
ALTER TABLE operations DROP CONSTRAINT IF EXISTS chk_operations_crew;
ALTER TABLE operations
    DROP COLUMN IF EXISTS run_time,
    DROP COLUMN IF EXISTS labor_time,
    DROP COLUMN IF EXISTS run_time_base_qty,
    DROP COLUMN IF EXISTS queue_time,
    DROP COLUMN IF EXISTS wait_time,
    DROP COLUMN IF EXISTS move_time,
    DROP COLUMN IF EXISTS crew_size,
    DROP COLUMN IF EXISTS time_unit;

COMMIT;
