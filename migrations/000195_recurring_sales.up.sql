CREATE TABLE IF NOT EXISTS recurring_sales_parameters (
    enterprise_code BIGINT PRIMARY KEY,
    current_month_billing_limit_day INTEGER NOT NULL DEFAULT 10,
    group_order_item_total BOOLEAN NOT NULL DEFAULT FALSE,
    indefinite_delivery_day INTEGER NOT NULL DEFAULT 10,
    fixed_term_delivery_day INTEGER NOT NULL DEFAULT 10,
    consider_discounts_additions BOOLEAN NOT NULL DEFAULT FALSE,
    generic_representative_code BIGINT,
    generic_sales_plan_code BIGINT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID NOT NULL,
    CONSTRAINT recurring_sales_param_days_chk CHECK (
        current_month_billing_limit_day BETWEEN 1 AND 31
        AND indefinite_delivery_day BETWEEN 1 AND 31
        AND fixed_term_delivery_day BETWEEN 1 AND 31
    )
);

CREATE TABLE IF NOT EXISTS recurring_sales_adjustment_dates (
    code BIGSERIAL PRIMARY KEY,
    enterprise_code BIGINT NOT NULL,
    customer_code BIGINT NOT NULL,
    establishment_code BIGINT,
    adjustment_date DATE NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    UNIQUE (enterprise_code, customer_code, establishment_code, adjustment_date)
);

CREATE TABLE IF NOT EXISTS recurring_sales (
    code BIGSERIAL PRIMARY KEY,
    enterprise_code BIGINT NOT NULL,
    customer_code BIGINT NOT NULL,
    establishment_code BIGINT,
    item_code BIGINT NOT NULL,
    item_mask VARCHAR(120),
    sales_plan_code BIGINT,
    movement_type VARCHAR(16) NOT NULL DEFAULT 'SALE',
    term_type VARCHAR(16) NOT NULL DEFAULT 'INDEFINITE',
    sale_date DATE NOT NULL,
    next_adjustment_date DATE,
    months_quantity INTEGER,
    payments_quantity INTEGER,
    grace_months INTEGER NOT NULL DEFAULT 0,
    payment_value NUMERIC(15,4),
    quantity NUMERIC(15,4) NOT NULL DEFAULT 1,
    unit_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    reason TEXT,
    generated_order_code BIGINT,
    generated_order_at TIMESTAMPTZ,
    source_recurring_sale_code BIGINT REFERENCES recurring_sales(code),
    original_adjustment_code BIGINT REFERENCES recurring_sales(code),
    adjustment_percent NUMERIC(9,4),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    CONSTRAINT recurring_sales_movement_chk CHECK (movement_type IN ('SALE','UPGRADE','DOWNGRADE','ADJUSTMENT','RECALCULATION','CANCELLATION')),
    CONSTRAINT recurring_sales_term_chk CHECK (term_type IN ('INDEFINITE','FIXED')),
    CONSTRAINT recurring_sales_indefinite_chk CHECK (term_type <> 'INDEFINITE' OR next_adjustment_date IS NOT NULL),
    CONSTRAINT recurring_sales_fixed_chk CHECK (
        term_type <> 'FIXED' OR (months_quantity IS NOT NULL AND months_quantity > 0 AND payments_quantity IS NOT NULL AND payments_quantity > 0 AND payment_value IS NOT NULL)
    ),
    CONSTRAINT recurring_sales_qty_chk CHECK (quantity > 0 AND unit_value >= 0),
    CONSTRAINT recurring_sales_grace_chk CHECK (grace_months >= 0)
);

CREATE INDEX IF NOT EXISTS idx_recurring_sales_console ON recurring_sales(customer_code, establishment_code, movement_type, is_active);
CREATE INDEX IF NOT EXISTS idx_recurring_sales_item ON recurring_sales(item_code, item_mask);
CREATE INDEX IF NOT EXISTS idx_recurring_sales_adjustment ON recurring_sales(next_adjustment_date);
CREATE INDEX IF NOT EXISTS idx_recurring_sales_order ON recurring_sales(generated_order_code);

CREATE TABLE IF NOT EXISTS recurring_sales_representatives (
    code BIGSERIAL PRIMARY KEY,
    recurring_sale_code BIGINT NOT NULL REFERENCES recurring_sales(code) ON DELETE CASCADE,
    representative_code BIGINT NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    commission_percent NUMERIC(9,4) NOT NULL DEFAULT 0,
    commission_base VARCHAR(16) NOT NULL DEFAULT 'ADJUSTED',
    is_lifetime BOOLEAN NOT NULL DEFAULT TRUE,
    commission_installments INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT recurring_sales_commission_base_chk CHECK (commission_base IN ('ORIGINAL','ADJUSTED')),
    CONSTRAINT recurring_sales_commission_installments_chk CHECK (is_lifetime OR (commission_installments IS NOT NULL AND commission_installments > 0)),
    CONSTRAINT recurring_sales_commission_percent_chk CHECK (commission_percent >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_recurring_sales_primary_rep
    ON recurring_sales_representatives(recurring_sale_code)
    WHERE is_primary;

CREATE TABLE IF NOT EXISTS recurring_sales_adjustment_links (
    adjustment_code BIGINT NOT NULL REFERENCES recurring_sales(code) ON DELETE CASCADE,
    source_recurring_sale_code BIGINT NOT NULL REFERENCES recurring_sales(code) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (adjustment_code, source_recurring_sale_code)
);
