BEGIN;

DROP INDEX IF EXISTS ix_fiscal_entries_supplier;
ALTER TABLE public.fiscal_entries DROP COLUMN IF EXISTS supplier_code;

COMMIT;
