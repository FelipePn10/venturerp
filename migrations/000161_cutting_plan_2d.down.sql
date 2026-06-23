BEGIN;

ALTER TABLE public.stock_remnants
    DROP COLUMN IF EXISTS width_mm,
    DROP COLUMN IF EXISTS height_mm;

ALTER TABLE public.cutting_pattern_placements
    DROP COLUMN IF EXISTS pos_x_mm,
    DROP COLUMN IF EXISTS pos_y_mm,
    DROP COLUMN IF EXISTS width_mm,
    DROP COLUMN IF EXISTS height_mm,
    DROP COLUMN IF EXISTS rotated;

ALTER TABLE public.cutting_patterns
    DROP COLUMN IF EXISTS stock_width_mm,
    DROP COLUMN IF EXISTS stock_height_mm,
    DROP COLUMN IF EXISTS used_area_mm2,
    DROP COLUMN IF EXISTS remnant_area_mm2,
    DROP COLUMN IF EXISTS remnant_width_mm,
    DROP COLUMN IF EXISTS remnant_height_mm;

ALTER TABLE public.cutting_stock_pieces
    DROP COLUMN IF EXISTS width_mm,
    DROP COLUMN IF EXISTS height_mm;

ALTER TABLE public.cutting_plan_parts
    DROP COLUMN IF EXISTS width_mm,
    DROP COLUMN IF EXISTS height_mm,
    DROP COLUMN IF EXISTS grain,
    DROP COLUMN IF EXISTS allow_rotation;

COMMIT;
