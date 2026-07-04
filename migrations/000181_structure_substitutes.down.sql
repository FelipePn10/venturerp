BEGIN;

DROP INDEX IF EXISTS idx_item_structures_subgroup;
ALTER TABLE item_structures
    DROP COLUMN IF EXISTS substitute_group,
    DROP COLUMN IF EXISTS substitute_priority;

COMMIT;
