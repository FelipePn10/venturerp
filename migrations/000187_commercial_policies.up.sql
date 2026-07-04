-- Comercial fase 2: commercial policy engine for discounts, surcharges,
-- freight and commissions.

CREATE TABLE IF NOT EXISTS public.commercial_policies (
    id                   BIGSERIAL PRIMARY KEY,
    code                 BIGINT NOT NULL UNIQUE,
    description          VARCHAR(150) NOT NULL,
    kind                 VARCHAR(20) NOT NULL,
    choice_type          VARCHAR(20) NOT NULL DEFAULT 'INFORMATION',
    calc_type            VARCHAR(20) NOT NULL DEFAULT 'PERCENT',
    percent_value        NUMERIC(10,4) NOT NULL DEFAULT 0,
    fixed_value          NUMERIC(15,4) NOT NULL DEFAULT 0,
    max_percent          NUMERIC(10,4) NOT NULL DEFAULT 0,
    max_value            NUMERIC(15,4) NOT NULL DEFAULT 0,
    min_gross_value      NUMERIC(15,4) NOT NULL DEFAULT 0,
    max_gross_value      NUMERIC(15,4) NOT NULL DEFAULT 0,
    min_quantity         NUMERIC(15,4) NOT NULL DEFAULT 0,
    max_quantity         NUMERIC(15,4) NOT NULL DEFAULT 0,
    priority             BIGINT NOT NULL DEFAULT 10,
    sequence             BIGINT NOT NULL DEFAULT 10,
    stackable            BOOLEAN NOT NULL DEFAULT TRUE,
    requires_approval    BOOLEAN NOT NULL DEFAULT FALSE,
    applies_on_net_value BOOLEAN NOT NULL DEFAULT FALSE,
    allow_manual_change  BOOLEAN NOT NULL DEFAULT FALSE,
    allow_higher_values  BOOLEAN NOT NULL DEFAULT FALSE,
    used_in_commission   BOOLEAN NOT NULL DEFAULT FALSE,
    applies_to_items     BOOLEAN NOT NULL DEFAULT FALSE,
    subtract_commission_base BOOLEAN NOT NULL DEFAULT FALSE,
    data_types_json      JSONB NOT NULL DEFAULT '[]'::jsonb,
    commission_discount_mode VARCHAR(20) NOT NULL DEFAULT 'REAL',
    customer_code        BIGINT,
    customer_type_id     BIGINT REFERENCES public.customer_types(id),
    market_segment_id    BIGINT REFERENCES public.market_segments(id),
    region_id            BIGINT REFERENCES public.regions(id),
    sales_table_id       BIGINT REFERENCES public.sales_tables(id),
    payment_condition_id BIGINT REFERENCES public.payment_conditions(id),
    carrier_id           BIGINT REFERENCES public.carriers(id),
    item_code            VARCHAR(60),
    item_mask            VARCHAR(60),
    product_line_id      BIGINT,
    item_classification  VARCHAR(80),
    rule_json            JSONB NOT NULL DEFAULT '{}'::jsonb,
    validity_start       DATE,
    validity_end         DATE,
    is_active            BOOLEAN NOT NULL DEFAULT TRUE,
    observation          TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_commercial_policies_kind
        CHECK (kind IN ('DISCOUNT', 'SURCHARGE', 'FREIGHT', 'COMMISSION')),
    CONSTRAINT chk_commercial_policies_choice_type
        CHECK (choice_type IN ('INFORMATION', 'CHOICE', 'OPTIONAL')),
    CONSTRAINT chk_commercial_policies_calc_type
        CHECK (calc_type IN ('PERCENT', 'VALUE')),
    CONSTRAINT chk_commercial_policies_commission_discount_mode
        CHECK (commission_discount_mode IN ('REAL', 'NOMINAL')),
    CONSTRAINT chk_commercial_policies_non_negative
        CHECK (
            percent_value >= 0 AND fixed_value >= 0 AND max_percent >= 0 AND max_value >= 0
            AND min_gross_value >= 0 AND max_gross_value >= 0
            AND min_quantity >= 0 AND max_quantity >= 0
        ),
    CONSTRAINT chk_commercial_policies_value_ranges
        CHECK (
            (max_gross_value = 0 OR min_gross_value <= max_gross_value)
            AND (max_quantity = 0 OR min_quantity <= max_quantity)
        )
);

CREATE INDEX IF NOT EXISTS idx_commercial_policies_active_kind
    ON public.commercial_policies(is_active, kind, priority, sequence);

CREATE INDEX IF NOT EXISTS idx_commercial_policies_context
    ON public.commercial_policies(customer_code, sales_table_id, item_code, product_line_id, item_classification);

CREATE TABLE IF NOT EXISTS public.commercial_policy_lines (
    id                  BIGSERIAL PRIMARY KEY,
    policy_id           BIGINT NOT NULL REFERENCES public.commercial_policies(id) ON DELETE CASCADE,
    line_number         BIGINT NOT NULL,
    sequence_number     BIGINT NOT NULL DEFAULT 1,
    description         VARCHAR(150),
    calc_type           VARCHAR(20) NOT NULL DEFAULT 'PERCENT',
    percent_value       NUMERIC(10,4) NOT NULL DEFAULT 0,
    fixed_value         NUMERIC(15,4) NOT NULL DEFAULT 0,
    min_value           NUMERIC(15,4) NOT NULL DEFAULT 0,
    max_value           NUMERIC(15,4) NOT NULL DEFAULT 0,
    variables_json      JSONB NOT NULL DEFAULT '{}'::jsonb,
    validity_start      DATE,
    validity_end        DATE,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_commercial_policy_lines_number UNIQUE (policy_id, line_number, sequence_number),
    CONSTRAINT chk_commercial_policy_lines_calc_type CHECK (calc_type IN ('PERCENT', 'VALUE')),
    CONSTRAINT chk_commercial_policy_lines_non_negative CHECK (
        percent_value >= 0 AND fixed_value >= 0 AND min_value >= 0 AND max_value >= 0
    ),
    CONSTRAINT chk_commercial_policy_lines_value_range CHECK (max_value = 0 OR min_value <= max_value)
);

CREATE INDEX IF NOT EXISTS idx_commercial_policy_lines_policy
    ON public.commercial_policy_lines(policy_id, is_active, line_number, sequence_number);

CREATE TABLE IF NOT EXISTS public.commercial_policy_specific_items (
    id                  BIGSERIAL PRIMARY KEY,
    policy_id           BIGINT NOT NULL REFERENCES public.commercial_policies(id) ON DELETE CASCADE,
    item_code           VARCHAR(60),
    item_mask           VARCHAR(60),
    product_line_id     BIGINT,
    item_classification VARCHAR(80),
    validity_start      DATE,
    validity_end        DATE,
    block_discount      BOOLEAN NOT NULL DEFAULT FALSE,
    block_surcharge     BOOLEAN NOT NULL DEFAULT FALSE,
    ignore_item_policies BOOLEAN NOT NULL DEFAULT FALSE,
    block_manual_change BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_commercial_policy_specific_items_target
        CHECK (item_code IS NOT NULL OR product_line_id IS NOT NULL OR item_classification IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_commercial_policy_specific_items_policy
    ON public.commercial_policy_specific_items(policy_id);
