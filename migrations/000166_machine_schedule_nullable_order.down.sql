BEGIN;
ALTER TABLE public.machine_schedules ALTER COLUMN order_code SET NOT NULL;
COMMIT;
