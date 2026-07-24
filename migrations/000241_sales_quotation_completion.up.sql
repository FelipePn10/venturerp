ALTER TABLE public.sales_divisions
    ADD COLUMN IF NOT EXISTS allow_free_payment_terms BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS public.sales_quotation_parameters (
    enterprise_code BIGINT PRIMARY KEY,
    purchase_order_prompt VARCHAR(80) NOT NULL DEFAULT 'Ordem de Compra',
    delivery_authorization_prompt VARCHAR(80) NOT NULL DEFAULT 'Autorização de Entr.',
    final_consumer_customer_code BIGINT,
    allow_service_items_nfce BOOLEAN NOT NULL DEFAULT FALSE,
    default_nfce BOOLEAN NOT NULL DEFAULT FALSE,
    minimum_cif_freight NUMERIC(18,4) NOT NULL DEFAULT 0,
    add_redelivery_to_freight BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_quotation_parameters_min_freight_chk CHECK (minimum_cif_freight >= 0)
);

CREATE TABLE IF NOT EXISTS public.sales_quotation_commission_patterns (
    id BIGSERIAL PRIMARY KEY,
    enterprise_code BIGINT NOT NULL,
    code BIGINT NOT NULL,
    description VARCHAR(160) NOT NULL,
    commission_pct NUMERIC(9,4) NOT NULL,
    invoice_pct NUMERIC(9,4) NOT NULL,
    payment_pct NUMERIC(9,4) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_quotation_commission_pattern_pct_chk CHECK (
        commission_pct >= 0 AND invoice_pct >= 0 AND payment_pct >= 0
        AND invoice_pct + payment_pct = commission_pct
    ),
    CONSTRAINT sales_quotation_commission_pattern_unique UNIQUE (enterprise_code, code)
);

CREATE TABLE IF NOT EXISTS public.sales_quotation_commission_pattern_sequences (
    enterprise_code BIGINT PRIMARY KEY,
    last_code BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS public.sales_quotation_cancellation_reasons (
    id BIGSERIAL PRIMARY KEY,
    enterprise_code BIGINT NOT NULL,
    code BIGINT NOT NULL,
    description VARCHAR(200) NOT NULL,
    allow_uncancel BOOLEAN NOT NULL DEFAULT FALSE,
    require_complement BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_quotation_cancellation_reason_unique UNIQUE (enterprise_code, code)
);

ALTER TABLE public.sales_quotations
    ADD COLUMN IF NOT EXISTS delivery_with_receipt BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS cancellation_reason_code BIGINT,
    ADD COLUMN IF NOT EXISTS dav_generated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS dav_report_key UUID,
    ADD COLUMN IF NOT EXISTS consumer_address VARCHAR(320);

ALTER TABLE public.sales_quotations DROP CONSTRAINT IF EXISTS sales_quotations_status_chk;
ALTER TABLE public.sales_quotations
    ADD CONSTRAINT sales_quotations_status_chk CHECK (
        status IN ('R','P','A','OA','F','OF','V','OV','CANCELLED','ATTENDED','EXPIRED')
    );
ALTER TABLE public.sales_quotations ALTER COLUMN status SET DEFAULT 'OV';

ALTER TABLE public.sales_quotation_events
    ADD COLUMN IF NOT EXISTS sales_quotation_item_code BIGINT;
ALTER TABLE public.sales_quotation_events
    DROP CONSTRAINT IF EXISTS sales_quotation_events_type_chk;
ALTER TABLE public.sales_quotation_events
    ADD CONSTRAINT sales_quotation_events_type_chk CHECK (
        event_type IN (
            'CANCEL','UNCANCEL','ATTEND','CONVERT','BLOCK','UNBLOCK',
            'RELEASE','MANUAL_RELEASE'
        )
    );

CREATE INDEX IF NOT EXISTS idx_sales_quotations_enterprise_list
    ON public.sales_quotations (enterprise_code, emission_date DESC, quotation_number DESC);
CREATE INDEX IF NOT EXISTS idx_sales_quotations_enterprise_customer
    ON public.sales_quotations (enterprise_code, customer_code);
CREATE INDEX IF NOT EXISTS idx_sales_quotation_commission_patterns_enterprise
    ON public.sales_quotation_commission_patterns (enterprise_code, is_active, code);
CREATE INDEX IF NOT EXISTS idx_sales_quotation_cancellation_reasons_enterprise
    ON public.sales_quotation_cancellation_reasons (enterprise_code, is_active, code);

ALTER TABLE public.sales_quotation_items
    DROP CONSTRAINT IF EXISTS sales_quotation_items_qty_chk;
ALTER TABLE public.sales_quotation_items
    ADD CONSTRAINT sales_quotation_items_qty_chk CHECK (
        requested_qty > 0 AND attended_qty >= 0 AND cancelled_qty >= 0
        AND attended_qty + cancelled_qty <= requested_qty
    );

ALTER TABLE public.sales_quotation_attachments
    DROP CONSTRAINT IF EXISTS sales_quotation_attachments_size_chk;
ALTER TABLE public.sales_quotation_attachments
    ADD COLUMN IF NOT EXISTS content BYTEA NOT NULL DEFAULT ''::bytea;
ALTER TABLE public.sales_quotation_attachments
    ADD CONSTRAINT sales_quotation_attachments_size_chk CHECK (
        file_size >= 0 AND file_size <= 10485760
    );
