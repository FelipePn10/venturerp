BEGIN;

DROP TABLE IF EXISTS public.purchase_order_attachments;
DROP TABLE IF EXISTS public.purchase_order_currency_rates;
DROP INDEX IF EXISTS public.idx_import_processes_purchase_order;
DROP INDEX IF EXISTS public.idx_purchase_order_items_consultation;
DROP INDEX IF EXISTS public.idx_purchase_orders_consultation_metadata;
DROP INDEX IF EXISTS public.idx_purchase_orders_consultation;

ALTER TABLE public.purchase_order_items
    DROP COLUMN IF EXISTS icms_st_value,
    DROP COLUMN IF EXISTS icms_st_base,
    DROP COLUMN IF EXISTS icms_value,
    DROP COLUMN IF EXISTS icms_base,
    DROP COLUMN IF EXISTS ipi_value,
    DROP COLUMN IF EXISTS ipi_base,
    DROP COLUMN IF EXISTS additions;

ALTER TABLE public.purchase_orders
    DROP CONSTRAINT IF EXISTS purchase_orders_order_type_chk,
    DROP COLUMN IF EXISTS customer_code,
    DROP COLUMN IF EXISTS kanban_origin,
    DROP COLUMN IF EXISTS order_type,
    DROP COLUMN IF EXISTS buyer_employee_code;

COMMIT;
