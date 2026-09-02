DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM public.sales_quotation_events
         WHERE event_type IN ('ITEM_CREATE','ITEM_UPDATE')
    ) THEN
        RAISE EXCEPTION 'rollback 000318 recusado: existem eventos de item que seriam perdidos';
    END IF;
END
$$;

ALTER TABLE public.sales_quotation_events
    DROP CONSTRAINT IF EXISTS sales_quotation_events_type_chk;

ALTER TABLE public.sales_quotation_events
    ADD CONSTRAINT sales_quotation_events_type_chk CHECK (
        event_type IN (
            'CANCEL','UNCANCEL','ATTEND','CONVERT','BLOCK','UNBLOCK',
            'RELEASE','MANUAL_RELEASE'
        )
    );
