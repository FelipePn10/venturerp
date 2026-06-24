BEGIN;

ALTER TABLE public.cutting_plans
    DROP COLUMN IF EXISTS stock_uom,
    DROP COLUMN IF EXISTS uom_factor;

COMMIT;
