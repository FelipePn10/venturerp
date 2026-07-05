-- Comercial fase 5: representantes.
-- Models representative types, representative cadastro tabs, report filters and
-- commercial follow-up without overloading customers/suppliers.

CREATE TABLE IF NOT EXISTS public.representative_types (
    code BIGSERIAL PRIMARY KEY,
    description VARCHAR(120) NOT NULL,
    is_free BOOLEAN NOT NULL DEFAULT FALSE,
    ignores_direct_billing BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.representatives (
    code BIGSERIAL PRIMARY KEY,
    is_customer BOOLEAN NOT NULL DEFAULT FALSE,
    customer_code BIGINT,
    is_supplier BOOLEAN NOT NULL DEFAULT FALSE,
    supplier_code BIGINT,
    name VARCHAR(180) NOT NULL,
    trade_name VARCHAR(120),
    type_code BIGINT REFERENCES public.representative_types(code),
    category_code BIGINT,
    register_date DATE NOT NULL DEFAULT CURRENT_DATE,
    core_number VARCHAR(40),
    document_number VARCHAR(32) NOT NULL,
    postal_code VARCHAR(12),
    city VARCHAR(100),
    state VARCHAR(2),
    full_address VARCHAR(220),
    street VARCHAR(160),
    street_number VARCHAR(30),
    complement VARCHAR(120),
    district VARCHAR(100),
    main_phone VARCHAR(40),
    main_email VARCHAR(160),
    device_quantity INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    blocked BOOLEAN NOT NULL DEFAULT FALSE,
    block_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_representatives_device_quantity CHECK (device_quantity >= 0),
    CONSTRAINT chk_representatives_state CHECK (state IS NULL OR length(state) = 2)
);

CREATE INDEX IF NOT EXISTS idx_representatives_type ON public.representatives(type_code);
CREATE INDEX IF NOT EXISTS idx_representatives_document ON public.representatives(document_number);
CREATE INDEX IF NOT EXISTS idx_representatives_state ON public.representatives(state);

CREATE TABLE IF NOT EXISTS public.representative_enterprises (
    id BIGSERIAL PRIMARY KEY,
    representative_code BIGINT NOT NULL REFERENCES public.representatives(code) ON DELETE CASCADE,
    enterprise_code BIGINT NOT NULL,
    enterprise_name VARCHAR(180),
    commission_pattern_code BIGINT,
    commission_pct NUMERIC(9,4) NOT NULL DEFAULT 0,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (representative_code, enterprise_code)
);

CREATE TABLE IF NOT EXISTS public.representative_accounting (
    id BIGSERIAL PRIMARY KEY,
    representative_code BIGINT NOT NULL REFERENCES public.representatives(code) ON DELETE CASCADE,
    enterprise_code BIGINT,
    event_type VARCHAR(20) NOT NULL,
    debit_account_code BIGINT,
    debit_cost_center_code BIGINT,
    credit_account_code BIGINT,
    credit_cost_center_code BIGINT,
    history_code BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_rep_accounting_event CHECK (event_type IN ('GENERATED','REVERSED')),
    UNIQUE (representative_code, enterprise_code, event_type)
);

CREATE TABLE IF NOT EXISTS public.representative_regions (
    id BIGSERIAL PRIMARY KEY,
    representative_code BIGINT NOT NULL REFERENCES public.representatives(code) ON DELETE CASCADE,
    enterprise_code BIGINT,
    region_code BIGINT NOT NULL,
    microregion_code BIGINT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rep_regions_region ON public.representative_regions(region_code);

CREATE TABLE IF NOT EXISTS public.representative_segments (
    id BIGSERIAL PRIMARY KEY,
    representative_code BIGINT NOT NULL REFERENCES public.representatives(code) ON DELETE CASCADE,
    enterprise_code BIGINT,
    microregion_code BIGINT,
    market_segment_code BIGINT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.representative_sales_plans (
    id BIGSERIAL PRIMARY KEY,
    representative_code BIGINT NOT NULL REFERENCES public.representatives(code) ON DELETE CASCADE,
    enterprise_code BIGINT,
    microregion_code BIGINT,
    sales_plan_code BIGINT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.representative_interests (
    id BIGSERIAL PRIMARY KEY,
    representative_code BIGINT NOT NULL REFERENCES public.representatives(code) ON DELETE CASCADE,
    item_classification_code BIGINT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (representative_code, item_classification_code)
);

CREATE TABLE IF NOT EXISTS public.representative_phones (
    id BIGSERIAL PRIMARY KEY,
    representative_code BIGINT NOT NULL REFERENCES public.representatives(code) ON DELETE CASCADE,
    ddi VARCHAR(5),
    ddd VARCHAR(5),
    phone VARCHAR(40) NOT NULL,
    phone_type VARCHAR(20) NOT NULL DEFAULT 'COMERCIAL',
    ranking INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_rep_phone_ranking CHECK (ranking > 0)
);

CREATE TABLE IF NOT EXISTS public.representative_emails (
    id BIGSERIAL PRIMARY KEY,
    representative_code BIGINT NOT NULL REFERENCES public.representatives(code) ON DELETE CASCADE,
    email VARCHAR(160) NOT NULL,
    ranking INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_rep_email_ranking CHECK (ranking > 0)
);

CREATE TABLE IF NOT EXISTS public.representative_correspondence_addresses (
    id BIGSERIAL PRIMARY KEY,
    representative_code BIGINT NOT NULL REFERENCES public.representatives(code) ON DELETE CASCADE,
    postal_code VARCHAR(12),
    city VARCHAR(100),
    state VARCHAR(2),
    full_address VARCHAR(220),
    street VARCHAR(160),
    street_number VARCHAR(30),
    complement VARCHAR(120),
    district VARCHAR(100),
    is_default BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_rep_correspondence_state CHECK (state IS NULL OR length(state) = 2)
);

CREATE TABLE IF NOT EXISTS public.representative_contacts (
    id BIGSERIAL PRIMARY KEY,
    representative_code BIGINT NOT NULL REFERENCES public.representatives(code) ON DELETE CASCADE,
    contact_type_code BIGINT,
    name VARCHAR(160) NOT NULL,
    role VARCHAR(100),
    phone VARCHAR(40),
    email VARCHAR(160),
    notes TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
