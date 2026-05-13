BEGIN;

-- =========================================================
-- ENSURE CODE IS UNIQUE
-- =========================================================

ALTER TABLE public.sales_order_demands
    ADD CONSTRAINT sales_order_demands_code_key
        UNIQUE (code);

-- =========================================================
-- ADD COLUMN
-- =========================================================

ALTER TABLE public.planned_orders
    ADD COLUMN sales_order_code BIGINT NULL;

-- =========================================================
-- BACKFILL DATA
-- =========================================================

UPDATE public.planned_orders po
SET sales_order_code = sod.code
    FROM public.sales_order_demands sod
WHERE po.sales_order_id = sod.id;

-- =========================================================
-- CREATE FK USING CODE
-- =========================================================

ALTER TABLE public.planned_orders
    ADD CONSTRAINT planned_orders_sales_order_code_fkey
        FOREIGN KEY (sales_order_code)
            REFERENCES public.sales_order_demands(code);

-- =========================================================
-- DROP OLD COLUMN
-- =========================================================

ALTER TABLE public.planned_orders
DROP COLUMN IF EXISTS sales_order_id;

-- =========================================================
-- INDEX
-- =========================================================

CREATE INDEX IF NOT EXISTS idx_planned_orders_sales_order_code
    ON public.planned_orders(sales_order_code);

COMMIT;