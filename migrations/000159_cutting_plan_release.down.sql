BEGIN;

ALTER TABLE public.cutting_stock_pieces
    DROP COLUMN IF EXISTS remnant_id,
    DROP COLUMN IF EXISTS heat_number;

ALTER TABLE public.cutting_plans
    DROP COLUMN IF EXISTS warehouse_id,
    DROP COLUMN IF EXISTS production_order_code,
    DROP COLUMN IF EXISTS lot_consumption_mode,
    DROP COLUMN IF EXISTS include_remnants,
    DROP COLUMN IF EXISTS released_at;

DROP TABLE IF EXISTS public.cutting_settings;
DROP TABLE IF EXISTS public.cutting_plan_consumptions;
DROP TABLE IF EXISTS public.stock_remnants;

COMMIT;
