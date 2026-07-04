BEGIN;

ALTER TABLE work_center_costs
    DROP COLUMN IF EXISTS machine_cost_per_hour,
    DROP COLUMN IF EXISTS labor_cost_per_hour;

COMMIT;
