BEGIN;

ALTER TABLE public.cutting_pattern_placements
    DROP COLUMN IF EXISTS rotation_deg;

ALTER TABLE public.cutting_plan_parts
    DROP COLUMN IF EXISTS geometry;

COMMIT;
