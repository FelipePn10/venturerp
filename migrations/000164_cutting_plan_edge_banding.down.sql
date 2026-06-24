BEGIN;
ALTER TABLE public.cutting_plan_parts
    DROP COLUMN IF EXISTS edge_top,
    DROP COLUMN IF EXISTS edge_bottom,
    DROP COLUMN IF EXISTS edge_left,
    DROP COLUMN IF EXISTS edge_right,
    DROP COLUMN IF EXISTS band_item_code,
    DROP COLUMN IF EXISTS band_cost_per_m;
COMMIT;
