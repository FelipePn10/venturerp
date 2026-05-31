BEGIN;

ALTER TABLE machine_types DROP COLUMN IF EXISTS requires_operator;

COMMIT;
