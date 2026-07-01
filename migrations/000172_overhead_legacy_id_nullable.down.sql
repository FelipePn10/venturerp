BEGIN;

-- Re-imposing NOT NULL requires the legacy id columns to be populated. Backfill
-- from the code columns before restoring the constraints.
UPDATE overhead_allocations oa
SET cost_center_id = cc.id
FROM cost_centers cc
WHERE oa.cost_center_code = cc.code AND oa.cost_center_id IS NULL;

UPDATE overhead_allocation_targets oat
SET overhead_id = oa.id
FROM overhead_allocations oa
WHERE oat.overhead_code = oa.code AND oat.overhead_id IS NULL;

UPDATE overhead_allocation_targets oat
SET cost_center_id = cc.id
FROM cost_centers cc
WHERE oat.cost_center_code = cc.code AND oat.cost_center_id IS NULL;

ALTER TABLE overhead_allocations        ALTER COLUMN cost_center_id  SET NOT NULL;
ALTER TABLE overhead_allocation_targets ALTER COLUMN overhead_id     SET NOT NULL;
ALTER TABLE overhead_allocation_targets ALTER COLUMN cost_center_id  SET NOT NULL;

COMMIT;
