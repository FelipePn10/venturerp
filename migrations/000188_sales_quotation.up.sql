CREATE TABLE IF NOT EXISTS public.sales_quotation_sequences (
    enterprise_code BIGINT PRIMARY KEY,
    last_number BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS public.sales_quotations (
    code BIGSERIAL PRIMARY KEY,
    quotation_number BIGINT NOT NULL,
    enterprise_code BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'OF',
    quotation_type VARCHAR(20) NOT NULL DEFAULT 'VENDA',
    emission_date DATE NOT NULL DEFAULT CURRENT_DATE,
    digit_date DATE NOT NULL DEFAULT CURRENT_DATE,
    valid_until DATE,
    delivery_date DATE,
    delivery_date_firm BOOLEAN NOT NULL DEFAULT FALSE,
    purchase_order_number VARCHAR(80),
    customer_code BIGINT,
    billing_address_code BIGINT,
    shipping_address_code BIGINT,
    representative_code BIGINT,
    sales_division_code BIGINT,
    price_table_code BIGINT,
    payment_term_code BIGINT,
    currency_code VARCHAR(10) NOT NULL DEFAULT 'BRL',
    probability_pct NUMERIC(7,4) NOT NULL DEFAULT 0,
    commission_pct NUMERIC(9,4) NOT NULL DEFAULT 0,
    is_nfce BOOLEAN NOT NULL DEFAULT FALSE,
    street VARCHAR(255),
    street_number VARCHAR(30),
    foreign_document VARCHAR(80),
    release_status VARCHAR(20) NOT NULL DEFAULT 'RELEASED',
    commercial_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    commercial_block_reason TEXT,
    carrier_code BIGINT,
    freight_type VARCHAR(30),
    verify_freight BOOLEAN NOT NULL DEFAULT FALSE,
    freight_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    redelivery_freight_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    insurance_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    discount_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    surcharge_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    retained_tax_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_gross NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_net NUMERIC(15,4) NOT NULL DEFAULT 0,
    delivery_authorization VARCHAR(120),
    notes TEXT,
    obs_customer TEXT,
    cancel_reason TEXT,
    cancel_complement TEXT,
    attended_reason TEXT,
    attended_at TIMESTAMPTZ,
    converted_sales_order_code BIGINT,
    converted_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    CONSTRAINT sales_quotations_status_chk CHECK (status IN ('R','P','A','OA','F','OF','CANCELLED','ATTENDED','EXPIRED')),
    CONSTRAINT sales_quotations_type_chk CHECK (quotation_type IN ('API_TERCEIROS','CONSULTA','FOCCOPORTAL','IMPORTADO','NEGOCIACAO','VENDA')),
    CONSTRAINT sales_quotations_release_chk CHECK (release_status IN ('BLOCKED','MANUAL_RELEASED','RELEASED')),
    CONSTRAINT sales_quotations_probability_chk CHECK (probability_pct >= 0 AND probability_pct <= 100),
    CONSTRAINT sales_quotations_unique_number UNIQUE (enterprise_code, quotation_number)
);

CREATE INDEX IF NOT EXISTS idx_sales_quotations_customer ON public.sales_quotations(customer_code);
CREATE INDEX IF NOT EXISTS idx_sales_quotations_status ON public.sales_quotations(status);
CREATE INDEX IF NOT EXISTS idx_sales_quotations_emission ON public.sales_quotations(emission_date);

CREATE TABLE IF NOT EXISTS public.sales_quotation_items (
    code BIGSERIAL PRIMARY KEY,
    sales_quotation_code BIGINT NOT NULL REFERENCES public.sales_quotations(code) ON DELETE CASCADE,
    sequence INT NOT NULL,
    item_code BIGINT NOT NULL,
    mask VARCHAR(80) NOT NULL DEFAULT '',
    sales_uom VARCHAR(20),
    warehouse_code BIGINT,
    price_table_code BIGINT,
    requested_qty NUMERIC(15,4) NOT NULL,
    unit_price NUMERIC(15,4) NOT NULL DEFAULT 0,
    attended_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
    cancelled_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
    delivery_date DATE,
    delivery_date_firm BOOLEAN NOT NULL DEFAULT FALSE,
    discount_pct NUMERIC(9,4) NOT NULL DEFAULT 0,
    ipi_pct NUMERIC(9,4) NOT NULL DEFAULT 0,
    st_pct NUMERIC(9,4) NOT NULL DEFAULT 0,
    total_gross NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_net NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_net_with_ipi NUMERIC(15,4) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN',
    notes TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_quotation_items_status_chk CHECK (status IN ('OPEN','PARTIAL','DELIVERED','CANCELLED')),
    CONSTRAINT sales_quotation_items_qty_chk CHECK (requested_qty > 0 AND attended_qty >= 0 AND cancelled_qty >= 0),
    CONSTRAINT sales_quotation_items_sequence_unique UNIQUE (sales_quotation_code, sequence)
);

CREATE INDEX IF NOT EXISTS idx_sales_quotation_items_header ON public.sales_quotation_items(sales_quotation_code);
CREATE INDEX IF NOT EXISTS idx_sales_quotation_items_item ON public.sales_quotation_items(item_code);

CREATE TABLE IF NOT EXISTS public.sales_quotation_events (
    id BIGSERIAL PRIMARY KEY,
    sales_quotation_code BIGINT NOT NULL REFERENCES public.sales_quotations(code) ON DELETE CASCADE,
    event_type VARCHAR(20) NOT NULL,
    reason TEXT NOT NULL,
    complement TEXT,
    event_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    CONSTRAINT sales_quotation_events_type_chk CHECK (event_type IN ('CANCEL','UNCANCEL','ATTEND','CONVERT','BLOCK','UNBLOCK'))
);

CREATE INDEX IF NOT EXISTS idx_sales_quotation_events_header ON public.sales_quotation_events(sales_quotation_code, created_at DESC);

CREATE TABLE IF NOT EXISTS public.sales_quotation_attachments (
    id BIGSERIAL PRIMARY KEY,
    sales_quotation_code BIGINT NOT NULL REFERENCES public.sales_quotations(code) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    content_type VARCHAR(120),
    file_size BIGINT NOT NULL DEFAULT 0,
    storage_key TEXT NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_by UUID,
    CONSTRAINT sales_quotation_attachments_size_chk CHECK (file_size <= 10485760)
);

CREATE INDEX IF NOT EXISTS idx_sales_quotation_attachments_header ON public.sales_quotation_attachments(sales_quotation_code);
