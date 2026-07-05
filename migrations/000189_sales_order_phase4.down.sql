DROP TABLE IF EXISTS public.sales_order_events;

DROP INDEX IF EXISTS public.idx_sales_orders_conference;
DROP INDEX IF EXISTS public.idx_sales_orders_release;
DROP INDEX IF EXISTS public.idx_sales_orders_analysis;

ALTER TABLE public.sales_orders
    DROP CONSTRAINT IF EXISTS sales_orders_conference_status_chk,
    DROP CONSTRAINT IF EXISTS sales_orders_release_status_chk,
    DROP CONSTRAINT IF EXISTS sales_orders_financial_analysis_chk,
    DROP CONSTRAINT IF EXISTS sales_orders_commercial_analysis_chk,
    DROP COLUMN IF EXISTS delay_action,
    DROP COLUMN IF EXISTS delay_reason,
    DROP COLUMN IF EXISTS attended_at,
    DROP COLUMN IF EXISTS attended_reason,
    DROP COLUMN IF EXISTS cancel_complement,
    DROP COLUMN IF EXISTS cancel_reason,
    DROP COLUMN IF EXISTS conference_status,
    DROP COLUMN IF EXISTS release_status,
    DROP COLUMN IF EXISTS financial_analysis_status,
    DROP COLUMN IF EXISTS commercial_analysis_status;
