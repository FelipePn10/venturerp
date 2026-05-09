ALTER TABLE public.planned_orders
    ADD COLUMN code BIGINT;

UPDATE public.planned_orders
SET code = id;

ALTER TABLE public.planned_orders
    ALTER COLUMN code SET NOT NULL;

ALTER TABLE public.planned_orders
    ADD CONSTRAINT planned_orders_code_key UNIQUE (code);

ALTER TABLE public.machines
    ADD COLUMN machine_type_code BIGINT,
ADD COLUMN cost_center_code BIGINT;

UPDATE public.machines m
SET machine_type_code = mt.code
    FROM public.machine_types mt
WHERE m.machine_type_id = mt.id;

UPDATE public.machines m
SET cost_center_code = cc.code
    FROM public.cost_centers cc
WHERE m.cost_center_id = cc.id;

ALTER TABLE public.machines
    ALTER COLUMN machine_type_code SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_machines_machine_type_code
    ON public.machines (machine_type_code);

CREATE INDEX IF NOT EXISTS idx_machines_cost_center_code
    ON public.machines (cost_center_code);

ALTER TABLE public.machines
    ADD CONSTRAINT machines_machine_type_code_fkey
        FOREIGN KEY (machine_type_code)
            REFERENCES public.machine_types (code);

ALTER TABLE public.machines
    ADD CONSTRAINT machines_cost_center_code_fkey
        FOREIGN KEY (cost_center_code)
            REFERENCES public.cost_centers (code);

ALTER TABLE public.machines
DROP CONSTRAINT machines_machine_type_id_fkey,
DROP CONSTRAINT machines_cost_center_id_fkey;

ALTER TABLE public.machines
DROP COLUMN machine_type_id,
DROP COLUMN cost_center_id;

ALTER TABLE public.machine_schedules
    ADD COLUMN machine_code BIGINT,
ADD COLUMN order_code BIGINT;

UPDATE public.machine_schedules ms
SET machine_code = m.code
    FROM public.machines m
WHERE ms.machine_id = m.id;

UPDATE public.machine_schedules ms
SET order_code = po.code
    FROM public.planned_orders po
WHERE ms.order_id = po.id;

ALTER TABLE public.machine_schedules
    ALTER COLUMN machine_code SET NOT NULL,
ALTER COLUMN order_code SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_machine_schedules_machine_code
    ON public.machine_schedules (machine_code);

CREATE INDEX IF NOT EXISTS idx_machine_schedules_order_code
    ON public.machine_schedules (order_code);

ALTER TABLE public.machine_schedules
    ADD CONSTRAINT machine_schedules_machine_code_fkey
        FOREIGN KEY (machine_code)
            REFERENCES public.machines (code);

ALTER TABLE public.machine_schedules
    ADD CONSTRAINT machine_schedules_order_code_fkey
        FOREIGN KEY (order_code)
            REFERENCES public.planned_orders (code);

ALTER TABLE public.machine_schedules
DROP CONSTRAINT machine_schedules_machine_id_fkey,
DROP CONSTRAINT machine_schedules_order_id_fkey;

ALTER TABLE public.machine_schedules
DROP COLUMN machine_id,
DROP COLUMN order_id;
