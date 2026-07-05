-- Comercial fase 6: metas de vendas.
-- Covers periods, representative goals, commercial group goals, carry-over
-- balances and goal reporting.

CREATE TABLE IF NOT EXISTS public.sales_goal_periods (
    code BIGSERIAL PRIMARY KEY,
    description VARCHAR(120) NOT NULL,
    period_type VARCHAR(10) NOT NULL DEFAULT 'MONTH',
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sales_goal_period_type CHECK (period_type IN ('MONTH','WEEK','CUSTOM')),
    CONSTRAINT chk_sales_goal_period_dates CHECK (end_date >= start_date)
);

CREATE INDEX IF NOT EXISTS idx_sales_goal_periods_dates
    ON public.sales_goal_periods(start_date, end_date);

CREATE TABLE IF NOT EXISTS public.sales_goals (
    code BIGSERIAL PRIMARY KEY,
    representative_code BIGINT NOT NULL,
    period_code BIGINT NOT NULL REFERENCES public.sales_goal_periods(code),
    analysis_base VARCHAR(12) NOT NULL,
    award_pct NUMERIC(9,4) NOT NULL DEFAULT 0,
    notes TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sales_goals_base CHECK (analysis_base IN ('SALES','INVOICING')),
    CONSTRAINT chk_sales_goals_award CHECK (award_pct >= 0),
    UNIQUE (representative_code, period_code, analysis_base)
);

CREATE INDEX IF NOT EXISTS idx_sales_goals_representative
    ON public.sales_goals(representative_code);

CREATE TABLE IF NOT EXISTS public.sales_goal_items (
    id BIGSERIAL PRIMARY KEY,
    goal_code BIGINT NOT NULL REFERENCES public.sales_goals(code) ON DELETE CASCADE,
    target_type VARCHAR(20) NOT NULL,
    item_code BIGINT,
    item_classification_code BIGINT,
    item_group_code BIGINT,
    sales_uom VARCHAR(10),
    target_quantity NUMERIC(15,4) NOT NULL DEFAULT 0,
    target_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    bonus_pct NUMERIC(9,4) NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sales_goal_items_type CHECK (target_type IN ('ITEM','CLASSIFICATION','GROUP')),
    CONSTRAINT chk_sales_goal_items_target CHECK (target_quantity >= 0 AND target_value >= 0 AND bonus_pct >= 0),
    CONSTRAINT chk_sales_goal_items_one_target CHECK (
        (target_type = 'ITEM' AND item_code IS NOT NULL AND item_classification_code IS NULL AND item_group_code IS NULL) OR
        (target_type = 'CLASSIFICATION' AND item_code IS NULL AND item_classification_code IS NOT NULL AND item_group_code IS NULL) OR
        (target_type = 'GROUP' AND item_code IS NULL AND item_classification_code IS NULL AND item_group_code IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_sales_goal_items_goal
    ON public.sales_goal_items(goal_code);

CREATE TABLE IF NOT EXISTS public.sales_goal_group_targets (
    id BIGSERIAL PRIMARY KEY,
    period_code BIGINT NOT NULL REFERENCES public.sales_goal_periods(code),
    commercial_group_code BIGINT NOT NULL,
    goal_type VARCHAR(12) NOT NULL,
    minimum_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    minimum_bonus_pct NUMERIC(9,4) NOT NULL DEFAULT 0,
    probable_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    probable_bonus_pct NUMERIC(9,4) NOT NULL DEFAULT 0,
    ideal_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    ideal_bonus_pct NUMERIC(9,4) NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sales_goal_group_type CHECK (goal_type IN ('SALES','INVOICING')),
    CONSTRAINT chk_sales_goal_group_values CHECK (
        minimum_value >= 0 AND probable_value >= 0 AND ideal_value >= 0 AND
        minimum_bonus_pct >= 0 AND probable_bonus_pct >= 0 AND ideal_bonus_pct >= 0
    ),
    UNIQUE (period_code, commercial_group_code, goal_type)
);

CREATE TABLE IF NOT EXISTS public.sales_goal_group_customers (
    id BIGSERIAL PRIMARY KEY,
    group_goal_id BIGINT NOT NULL REFERENCES public.sales_goal_group_targets(id) ON DELETE CASCADE,
    customer_code BIGINT NOT NULL,
    representative_code BIGINT,
    minimum_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    minimum_bonus_pct NUMERIC(9,4) NOT NULL DEFAULT 0,
    probable_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    probable_bonus_pct NUMERIC(9,4) NOT NULL DEFAULT 0,
    ideal_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    ideal_bonus_pct NUMERIC(9,4) NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (group_goal_id, customer_code)
);

CREATE INDEX IF NOT EXISTS idx_sales_goal_group_customers_customer
    ON public.sales_goal_group_customers(customer_code);

CREATE TABLE IF NOT EXISTS public.sales_goal_balances (
    id BIGSERIAL PRIMARY KEY,
    period_code BIGINT NOT NULL REFERENCES public.sales_goal_periods(code),
    next_period_code BIGINT REFERENCES public.sales_goal_periods(code),
    balance_scope VARCHAR(20) NOT NULL,
    representative_code BIGINT,
    commercial_group_code BIGINT,
    customer_code BIGINT,
    goal_type VARCHAR(12) NOT NULL,
    realized_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    ideal_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    balance_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sales_goal_balance_scope CHECK (balance_scope IN ('REPRESENTATIVE','GROUP','CUSTOMER')),
    CONSTRAINT chk_sales_goal_balance_type CHECK (goal_type IN ('SALES','INVOICING')),
    CONSTRAINT chk_sales_goal_balance_values CHECK (realized_value >= 0 AND ideal_value >= 0 AND balance_value >= 0)
);

CREATE INDEX IF NOT EXISTS idx_sales_goal_balances_period
    ON public.sales_goal_balances(period_code, balance_scope);
