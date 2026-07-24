ALTER TABLE public.sales_quotation_attachments
    DROP CONSTRAINT IF EXISTS sales_quotation_attachments_size_chk;
ALTER TABLE public.sales_quotation_attachments
    DROP COLUMN IF EXISTS content;
ALTER TABLE public.sales_quotation_attachments
    ADD CONSTRAINT sales_quotation_attachments_size_chk CHECK (file_size <= 10485760);

ALTER TABLE public.sales_quotation_items
    DROP CONSTRAINT IF EXISTS sales_quotation_items_qty_chk;
ALTER TABLE public.sales_quotation_items
    ADD CONSTRAINT sales_quotation_items_qty_chk CHECK (
        requested_qty > 0 AND attended_qty >= 0 AND cancelled_qty >= 0
    );

DROP INDEX IF EXISTS public.idx_sales_quotation_cancellation_reasons_enterprise;
DROP INDEX IF EXISTS public.idx_sales_quotation_commission_patterns_enterprise;
DROP INDEX IF EXISTS public.idx_sales_quotations_enterprise_customer;
DROP INDEX IF EXISTS public.idx_sales_quotations_enterprise_list;

ALTER TABLE public.sales_quotations
    DROP COLUMN IF EXISTS consumer_address,
    DROP COLUMN IF EXISTS dav_report_key,
    DROP COLUMN IF EXISTS dav_generated_at,
    DROP COLUMN IF EXISTS cancellation_reason_code,
    DROP COLUMN IF EXISTS delivery_with_receipt;

ALTER TABLE public.sales_quotation_events
    DROP COLUMN IF EXISTS sales_quotation_item_code;
UPDATE public.sales_quotation_events
SET event_type = 'UNBLOCK'
WHERE event_type IN ('RELEASE', 'MANUAL_RELEASE');
ALTER TABLE public.sales_quotation_events
    DROP CONSTRAINT IF EXISTS sales_quotation_events_type_chk;
ALTER TABLE public.sales_quotation_events
    ADD CONSTRAINT sales_quotation_events_type_chk CHECK (
        event_type IN ('CANCEL','UNCANCEL','ATTEND','CONVERT','BLOCK','UNBLOCK')
    );

ALTER TABLE public.sales_quotations ALTER COLUMN status SET DEFAULT 'OF';
ALTER TABLE public.sales_quotations DROP CONSTRAINT IF EXISTS sales_quotations_status_chk;
UPDATE public.sales_quotations SET status='F' WHERE status='V';
UPDATE public.sales_quotations SET status='OF' WHERE status='OV';
ALTER TABLE public.sales_quotations
    ADD CONSTRAINT sales_quotations_status_chk CHECK (
        status IN ('R','P','A','OA','F','OF','CANCELLED','ATTENDED','EXPIRED')
    );

DROP TABLE IF EXISTS public.sales_quotation_cancellation_reasons;
DROP TABLE IF EXISTS public.sales_quotation_commission_pattern_sequences;
DROP TABLE IF EXISTS public.sales_quotation_commission_patterns;
DROP TABLE IF EXISTS public.sales_quotation_parameters;

ALTER TABLE public.sales_divisions
    DROP COLUMN IF EXISTS allow_free_payment_terms;
