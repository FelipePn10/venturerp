-- Comercial fase 1: pricing policy, table repricing and price history.

CREATE TABLE IF NOT EXISTS public.sales_price_policies (
    id               BIGSERIAL PRIMARY KEY,
    code             BIGINT NOT NULL UNIQUE,
    description      VARCHAR(150) NOT NULL,
    cost_source      VARCHAR(40) NOT NULL DEFAULT 'STANDARD_TOTAL',
    priority         BIGINT NOT NULL DEFAULT 10,
    sequence         BIGINT NOT NULL DEFAULT 10,
    policy_scope     VARCHAR(10) NOT NULL DEFAULT 'PREC',
    policy_types     TEXT NOT NULL DEFAULT '',
    markup_pct       NUMERIC(8,4) NOT NULL DEFAULT 0,
    margin_pct       NUMERIC(8,4) NOT NULL DEFAULT 0,
    max_margin_pct   NUMERIC(8,4) NOT NULL DEFAULT 0,
    ideal_margin_pct NUMERIC(8,4) NOT NULL DEFAULT 0,
    margin_step_pct  NUMERIC(8,4) NOT NULL DEFAULT 0,
    expenses_pct     NUMERIC(8,4) NOT NULL DEFAULT 0,
    taxes_pct        NUMERIC(8,4) NOT NULL DEFAULT 0,
    freight_pct      NUMERIC(8,4) NOT NULL DEFAULT 0,
    commission_pct   NUMERIC(8,4) NOT NULL DEFAULT 0,
    discount_pct     NUMERIC(8,4) NOT NULL DEFAULT 0,
    min_margin_pct   NUMERIC(8,4) NOT NULL DEFAULT 0,
    max_discount_pct NUMERIC(8,4) NOT NULL DEFAULT 0,
    incidences_json  JSONB NOT NULL DEFAULT '[]'::jsonb,
    sales_table_id   BIGINT REFERENCES public.sales_tables(id),
    validity_start   DATE,
    validity_end     DATE,
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    observation      TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sales_price_policy_cost_source
        CHECK (cost_source IN ('INFORMED', 'STANDARD_TOTAL', 'STANDARD_MATERIAL',
                               'PURCHASE', 'STOCK_AVG', 'STOCK_LAST')),
    CONSTRAINT chk_sales_price_policy_scope
        CHECK (policy_scope IN ('FPPV', 'PREC')),
    CONSTRAINT chk_sales_price_policy_margin_range
        CHECK (min_margin_pct <= max_margin_pct OR max_margin_pct = 0)
);

CREATE INDEX IF NOT EXISTS idx_sales_price_policies_active
    ON public.sales_price_policies(is_active, validity_start, validity_end);

CREATE UNIQUE INDEX IF NOT EXISTS uq_sales_price_policies_scope_priority_sequence_period
    ON public.sales_price_policies(policy_scope, priority, sequence, COALESCE(validity_start, DATE '0001-01-01'), COALESCE(validity_end, DATE '9999-12-31'));

CREATE TABLE IF NOT EXISTS public.sales_table_price_history (
    id                   BIGSERIAL PRIMARY KEY,
    sales_table_price_id BIGINT REFERENCES public.sales_table_prices(id) ON DELETE SET NULL,
    sales_table_id       BIGINT NOT NULL REFERENCES public.sales_tables(id),
    sales_table_code     BIGINT NOT NULL,
    item_code            VARCHAR(60) NOT NULL,
    old_price            NUMERIC(15,4),
    new_price            NUMERIC(15,4) NOT NULL,
    base_cost            NUMERIC(20,6),
    source               VARCHAR(40) NOT NULL DEFAULT 'MANUAL',
    policy_code          BIGINT,
    reason               TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sales_price_history_table_item
    ON public.sales_table_price_history(sales_table_code, item_code, created_at DESC);
