BEGIN;

ALTER TABLE public.fiscal_exit_items
    DROP COLUMN IF EXISTS aliq_pis,
    DROP COLUMN IF EXISTS aliq_cofins;

COMMIT;
