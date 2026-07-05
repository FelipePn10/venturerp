ALTER TABLE public.sales_orders
    ADD COLUMN IF NOT EXISTS commercial_analysis_status VARCHAR(20) NOT NULL DEFAULT 'NOT_ANALYZED',
    ADD COLUMN IF NOT EXISTS financial_analysis_status VARCHAR(20) NOT NULL DEFAULT 'NOT_ANALYZED',
    ADD COLUMN IF NOT EXISTS release_status VARCHAR(20) NOT NULL DEFAULT 'RELEASED',
    ADD COLUMN IF NOT EXISTS conference_status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    ADD COLUMN IF NOT EXISTS cancel_reason TEXT,
    ADD COLUMN IF NOT EXISTS cancel_complement TEXT,
    ADD COLUMN IF NOT EXISTS attended_reason TEXT,
    ADD COLUMN IF NOT EXISTS attended_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS delay_reason TEXT,
    ADD COLUMN IF NOT EXISTS delay_action TEXT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'sales_orders_commercial_analysis_chk'
    ) THEN
        ALTER TABLE public.sales_orders
            ADD CONSTRAINT sales_orders_commercial_analysis_chk
            CHECK (commercial_analysis_status IN ('NOT_ANALYZED','APPROVED','REJECTED'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'sales_orders_financial_analysis_chk'
    ) THEN
        ALTER TABLE public.sales_orders
            ADD CONSTRAINT sales_orders_financial_analysis_chk
            CHECK (financial_analysis_status IN ('NOT_ANALYZED','APPROVED','REJECTED'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'sales_orders_release_status_chk'
    ) THEN
        ALTER TABLE public.sales_orders
            ADD CONSTRAINT sales_orders_release_status_chk
            CHECK (release_status IN ('BLOCKED','MANUAL_RELEASED','RELEASED'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'sales_orders_conference_status_chk'
    ) THEN
        ALTER TABLE public.sales_orders
            ADD CONSTRAINT sales_orders_conference_status_chk
            CHECK (conference_status IN ('PENDING','CONFERRED','DIVERGENT'));
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS public.sales_order_events (
    id BIGSERIAL PRIMARY KEY,
    sales_order_code BIGINT NOT NULL REFERENCES public.sales_orders(code) ON DELETE CASCADE,
    event_type VARCHAR(30) NOT NULL,
    area VARCHAR(20),
    reason TEXT,
    complement TEXT,
    event_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    CONSTRAINT sales_order_events_type_chk CHECK (
        event_type IN ('ANALYZE','RELEASE','BLOCK','UNBLOCK','CANCEL','ATTEND','CONFER','DELAY_REASON')
    ),
    CONSTRAINT sales_order_events_area_chk CHECK (
        area IS NULL OR area IN ('COMMERCIAL','FINANCIAL','ENGINEERING','LOGISTICS')
    )
);

CREATE INDEX IF NOT EXISTS idx_sales_orders_analysis
    ON public.sales_orders(commercial_analysis_status, financial_analysis_status);

CREATE INDEX IF NOT EXISTS idx_sales_orders_release
    ON public.sales_orders(release_status, is_blocked);

CREATE INDEX IF NOT EXISTS idx_sales_orders_conference
    ON public.sales_orders(conference_status);

CREATE INDEX IF NOT EXISTS idx_sales_order_events_order
    ON public.sales_order_events(sales_order_code, created_at DESC);
