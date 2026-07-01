BEGIN;

-- The item_machine_times and machine_schedules queries (list/get, the
-- production-time calculator and the soft-delete DeleteSchedule) all filter on
-- an is_active flag that was never added to either table, raising
-- 'column "is_active" does not exist' (SQLSTATE 42703). Add it with a TRUE
-- default so existing rows remain visible.
ALTER TABLE public.item_machine_times
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE public.machine_schedules
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

COMMIT;
