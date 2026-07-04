BEGIN;

DROP INDEX IF EXISTS idx_mfg_routes_validity;
ALTER TABLE manufacturing_routes DROP CONSTRAINT IF EXISTS chk_mfg_routes_validity;
ALTER TABLE manufacturing_routes
    DROP COLUMN IF EXISTS valid_from,
    DROP COLUMN IF EXISTS valid_to;

COMMIT;
