BEGIN;

ALTER TABLE route_operations
    DROP COLUMN IF EXISTS supplier_id,
    DROP COLUMN IF EXISTS service_item_code,
    DROP COLUMN IF EXISTS cost_per_unit,
    DROP COLUMN IF EXISTS lead_time_days;

ALTER TABLE operations
    DROP COLUMN IF EXISTS supplier_id,
    DROP COLUMN IF EXISTS service_item_code,
    DROP COLUMN IF EXISTS cost_per_unit,
    DROP COLUMN IF EXISTS lead_time_days;

COMMIT;
