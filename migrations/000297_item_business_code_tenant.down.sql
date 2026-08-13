BEGIN;

ALTER TABLE items
    DROP CONSTRAINT IF EXISTS fk_items_enterprise,
    DROP CONSTRAINT IF EXISTS chk_items_business_code_format,
    DROP CONSTRAINT IF EXISTS uq_items_enterprise_business_code;

DROP INDEX IF EXISTS idx_items_enterprise_business_code;

ALTER TABLE items
    ALTER COLUMN code DROP DEFAULT,
    DROP COLUMN IF EXISTS business_code,
    DROP COLUMN IF EXISTS enterprise_id;

DROP SEQUENCE IF EXISTS items_legacy_code_seq;

COMMIT;
