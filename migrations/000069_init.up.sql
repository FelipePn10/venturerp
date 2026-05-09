ALTER TABLE public.machine_schedules
    ADD COLUMN code BIGINT;

UPDATE public.machine_schedules
SET code = id;

ALTER TABLE public.machine_schedules
    ALTER COLUMN code SET NOT NULL;

ALTER TABLE public.machine_schedules
    ADD CONSTRAINT machine_schedules_code_key UNIQUE (code);

CREATE INDEX IF NOT EXISTS idx_machine_schedules_code
    ON public.machine_schedules (code);

COMMIT;