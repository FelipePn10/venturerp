ALTER TABLE public.sales_quotation_events
    DROP CONSTRAINT IF EXISTS sales_quotation_events_type_chk;

ALTER TABLE public.sales_quotation_events
    ADD CONSTRAINT sales_quotation_events_type_chk CHECK (
        event_type IN (
            'CANCEL','UNCANCEL','ATTEND','CONVERT','BLOCK','UNBLOCK',
            'RELEASE','MANUAL_RELEASE','ITEM_CREATE','ITEM_UPDATE'
        )
    );
