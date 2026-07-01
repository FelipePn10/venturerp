BEGIN;

-- The overhead-allocation module migrated to code-based columns
-- (cost_center_code / overhead_code) in migrations 000083-000085, but the
-- original NOT NULL id columns were left in place. INSERTs only populate the
-- *_code columns, so the legacy *_id columns raised
-- 'null value in column "cost_center_id"' (SQLSTATE 23502). Drop the NOT NULL
-- constraints so the code-based inserts succeed; the id columns remain for
-- backward compatibility but are no longer required.
ALTER TABLE overhead_allocations        ALTER COLUMN cost_center_id  DROP NOT NULL;
ALTER TABLE overhead_allocation_targets ALTER COLUMN overhead_id     DROP NOT NULL;
ALTER TABLE overhead_allocation_targets ALTER COLUMN cost_center_id  DROP NOT NULL;

COMMIT;
