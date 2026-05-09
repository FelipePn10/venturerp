ALTER TABLE public.item_machine_times
    ADD COLUMN code BIGINT;

UPDATE public.item_machine_times
SET code = id;

ALTER TABLE public.item_machine_times
    ALTER COLUMN code SET NOT NULL;

ALTER TABLE public.item_machine_times
    ADD CONSTRAINT item_machine_times_code_key UNIQUE (code);

CREATE INDEX IF NOT EXISTS idx_item_machine_times_code
    ON public.item_machine_times (code);
