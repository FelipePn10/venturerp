BEGIN;
DROP INDEX IF EXISTS public.idx_delivery_reschedules_batch;
DROP INDEX IF EXISTS public.idx_delivery_reschedules_tenant_order;
ALTER TABLE public.delivery_reschedules DROP COLUMN IF EXISTS batch_id, DROP COLUMN IF EXISTS enterprise_code;
DROP TABLE IF EXISTS public.delivery_reschedule_batches;
DROP INDEX IF EXISTS public.ux_sales_orders_tenant_code;
COMMIT;
