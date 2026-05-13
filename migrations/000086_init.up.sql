BEGIN;

-- =========================================================
-- ADD CODE COLUMNS (SOURCE OF TRUTH)
-- =========================================================

ALTER TABLE public.planned_orders
    ADD COLUMN machine_code BIGINT NULL,
    ADD COLUMN cost_center_code BIGINT NULL,
    ADD COLUMN employee_code BIGINT NULL,
    ADD COLUMN parent_order_code BIGINT NULL,
    ADD COLUMN plan_code BIGINT NULL;

-- =========================================================
-- BACKFILL DATA FROM IDs -> CODES
-- =========================================================

UPDATE public.planned_orders po
SET machine_code = m.code
    FROM public.machines m
WHERE po.machine_id = m.id;

UPDATE public.planned_orders po
SET cost_center_code = cc.code
    FROM public.cost_centers cc
WHERE po.cost_center_id = cc.id;

UPDATE public.planned_orders po
SET employee_code = e.code
    FROM public.employees e
WHERE po.employee_id = e.id;

UPDATE public.planned_orders po
SET parent_order_code = parent.code
    FROM public.planned_orders parent
WHERE po.parent_order_id = parent.id;

UPDATE public.planned_orders po
SET plan_code = pp.code
    FROM public.production_plans pp
WHERE po.plan_id = pp.id;

-- =========================================================
-- CREATE FOREIGN KEYS USING CODE
-- =========================================================

ALTER TABLE public.planned_orders
    ADD CONSTRAINT planned_orders_machine_code_fkey
        FOREIGN KEY (machine_code)
            REFERENCES public.machines(code),

    ADD CONSTRAINT planned_orders_cost_center_code_fkey
        FOREIGN KEY (cost_center_code)
        REFERENCES public.cost_centers(code),

    ADD CONSTRAINT planned_orders_employee_code_fkey
        FOREIGN KEY (employee_code)
        REFERENCES public.employees(code),

    ADD CONSTRAINT planned_orders_parent_order_code_fkey
        FOREIGN KEY (parent_order_code)
        REFERENCES public.planned_orders(code),

    ADD CONSTRAINT planned_orders_plan_code_fkey
        FOREIGN KEY (plan_code)
        REFERENCES public.production_plans(code);

-- =========================================================
-- DROP OLD FOREIGN KEYS
-- =========================================================

ALTER TABLE public.planned_orders
DROP CONSTRAINT IF EXISTS planned_orders_machine_id_fkey,
    DROP CONSTRAINT IF EXISTS planned_orders_cost_center_id_fkey,
    DROP CONSTRAINT IF EXISTS planned_orders_employee_id_fkey,
    DROP CONSTRAINT IF EXISTS planned_orders_parent_order_id_fkey,
    DROP CONSTRAINT IF EXISTS planned_orders_plan_id_fkey;

-- =========================================================
-- DROP OLD ID COLUMNS
-- =========================================================

ALTER TABLE public.planned_orders
DROP COLUMN IF EXISTS machine_id,
    DROP COLUMN IF EXISTS cost_center_id,
    DROP COLUMN IF EXISTS employee_id,
    DROP COLUMN IF EXISTS parent_order_id,
    DROP COLUMN IF EXISTS plan_id;

-- =========================================================
-- ENSURE CODE IS BIGSERIAL + NOT NULL
-- =========================================================

CREATE SEQUENCE IF NOT EXISTS public.planned_orders_code_seq;

ALTER SEQUENCE public.planned_orders_code_seq
    OWNED BY public.planned_orders.code;

SELECT setval(
               'public.planned_orders_code_seq',
               COALESCE((SELECT MAX(code) FROM public.planned_orders), 1)
       );

ALTER TABLE public.planned_orders
    ALTER COLUMN code SET DEFAULT nextval('public.planned_orders_code_seq'),
ALTER COLUMN code SET NOT NULL;

-- =========================================================
-- INDEXES
-- =========================================================

CREATE INDEX IF NOT EXISTS idx_planned_orders_machine_code
    ON public.planned_orders(machine_code);

CREATE INDEX IF NOT EXISTS idx_planned_orders_cost_center_code
    ON public.planned_orders(cost_center_code);

CREATE INDEX IF NOT EXISTS idx_planned_orders_employee_code
    ON public.planned_orders(employee_code);

CREATE INDEX IF NOT EXISTS idx_planned_orders_parent_order_code
    ON public.planned_orders(parent_order_code);

CREATE INDEX IF NOT EXISTS idx_planned_orders_plan_code
    ON public.planned_orders(plan_code);

COMMIT;