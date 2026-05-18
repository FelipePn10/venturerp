BEGIN;

ALTER TABLE restrictions DROP CONSTRAINT IF EXISTS fk_restrictions_reason;
ALTER TABLE restrictions DROP COLUMN IF EXISTS customer_code;
DROP TABLE IF EXISTS restriction_reasons;

COMMIT;
