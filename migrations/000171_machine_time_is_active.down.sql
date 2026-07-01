BEGIN;

ALTER TABLE public.machine_schedules DROP COLUMN IF EXISTS is_active;
ALTER TABLE public.item_machine_times DROP COLUMN IF EXISTS is_active;

COMMIT;
