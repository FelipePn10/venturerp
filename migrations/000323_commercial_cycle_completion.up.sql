ALTER TABLE recurring_sales
    ADD COLUMN IF NOT EXISTS lifecycle_status VARCHAR(20) NOT NULL DEFAULT 'ATIVA',
    ADD COLUMN IF NOT EXISTS effective_from DATE,
    ADD COLUMN IF NOT EXISTS effective_until DATE,
    ADD COLUMN IF NOT EXISTS frequency VARCHAR(20) NOT NULL DEFAULT 'MENSAL',
    ADD COLUMN IF NOT EXISTS price_table_code BIGINT,
    ADD COLUMN IF NOT EXISTS currency_code VARCHAR(3) NOT NULL DEFAULT 'BRL',
    ADD COLUMN IF NOT EXISTS adjustment_index VARCHAR(40),
    ADD COLUMN IF NOT EXISTS adjustment_period_months INTEGER,
    ADD COLUMN IF NOT EXISTS adjustment_floor_pct NUMERIC(9,4),
    ADD COLUMN IF NOT EXISTS adjustment_cap_pct NUMERIC(9,4),
    ADD COLUMN IF NOT EXISTS billing_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS delivery_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS tax_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS cost_center_code BIGINT,
    ADD COLUMN IF NOT EXISTS renewal_policy VARCHAR(20) NOT NULL DEFAULT 'AUTOMATICA',
    ADD COLUMN IF NOT EXISTS cancellation_effective_date DATE,
    ADD COLUMN IF NOT EXISTS future_orders_policy VARCHAR(30),
    ADD COLUMN IF NOT EXISTS cancelled_by UUID;

ALTER TABLE recurring_sales DROP CONSTRAINT IF EXISTS recurring_sales_lifecycle_status_chk;
ALTER TABLE recurring_sales ADD CONSTRAINT recurring_sales_lifecycle_status_chk
    CHECK (lifecycle_status IN ('ATIVA','SUSPENSA','CANCELADA','ENCERRADA'));
ALTER TABLE recurring_sales DROP CONSTRAINT IF EXISTS recurring_sales_frequency_chk;
ALTER TABLE recurring_sales ADD CONSTRAINT recurring_sales_frequency_chk
    CHECK (frequency IN ('SEMANAL','QUINZENAL','MENSAL','BIMESTRAL','TRIMESTRAL','SEMESTRAL','ANUAL'));
ALTER TABLE recurring_sales DROP CONSTRAINT IF EXISTS recurring_sales_future_orders_policy_chk;
ALTER TABLE recurring_sales ADD CONSTRAINT recurring_sales_future_orders_policy_chk
    CHECK (future_orders_policy IS NULL OR future_orders_policy IN ('MANTER','CANCELAR_NAO_FATURADOS','NAO_GERAR_NOVOS'));

UPDATE recurring_sales
SET effective_from=COALESCE(effective_from, sale_date),
    lifecycle_status=CASE WHEN is_active THEN 'ATIVA' ELSE 'CANCELADA' END
WHERE effective_from IS NULL OR lifecycle_status='ATIVA' AND NOT is_active;

CREATE UNIQUE INDEX IF NOT EXISTS ux_recurring_sales_tenant_code
    ON recurring_sales(enterprise_code, code);

CREATE TABLE recurring_sales_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_code BIGINT NOT NULL REFERENCES enterprise(code),
    recurring_sale_code BIGINT NOT NULL,
    event_type VARCHAR(40) NOT NULL,
    before_state JSONB,
    after_state JSONB NOT NULL,
    reason TEXT,
    correlation_id VARCHAR(120),
    actor_id UUID NOT NULL REFERENCES users(id),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (enterprise_code, recurring_sale_code)
        REFERENCES recurring_sales(enterprise_code, code)
);
CREATE INDEX idx_recurring_sales_events_tenant_aggregate
    ON recurring_sales_events(enterprise_code, recurring_sale_code, occurred_at DESC);

CREATE TABLE recurring_sales_operations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_code BIGINT NOT NULL REFERENCES enterprise(code),
    recurring_sale_code BIGINT NOT NULL,
    operation_type VARCHAR(20) NOT NULL CHECK (operation_type IN ('PEDIDO','REAJUSTE')),
    competence VARCHAR(7) NOT NULL CHECK (competence ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
    idempotency_key VARCHAR(160) NOT NULL,
    request_hash VARCHAR(64) NOT NULL,
    result_code BIGINT,
    status VARCHAR(20) NOT NULL DEFAULT 'EM_PROCESSAMENTO' CHECK (status IN ('EM_PROCESSAMENTO','CONCLUIDA','FALHOU')),
    actor_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    UNIQUE(enterprise_code, recurring_sale_code, operation_type, competence),
    UNIQUE(enterprise_code, idempotency_key),
    FOREIGN KEY (enterprise_code, recurring_sale_code)
        REFERENCES recurring_sales(enterprise_code, code)
);
CREATE INDEX idx_recurring_sales_operations_tenant_status
    ON recurring_sales_operations(enterprise_code, status, created_at);

CREATE TABLE technical_assistance_rma_evidences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    rma_code BIGINT NOT NULL REFERENCES technical_assistance_rmas(code),
    file_name VARCHAR(255) NOT NULL,
    content_type VARCHAR(120) NOT NULL,
    content BYTEA NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (
        size_bytes > 0
        AND size_bytes <= 10485760
        AND size_bytes = octet_length(content)
    ),
    sha256 VARCHAR(64) NOT NULL,
    uploaded_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(enterprise_id, rma_code, sha256),
    FOREIGN KEY (enterprise_id, rma_code)
        REFERENCES technical_assistance_rmas(enterprise_id, code)
);
CREATE INDEX idx_rma_evidences_tenant_rma
    ON technical_assistance_rma_evidences(enterprise_id, rma_code, created_at DESC);

ALTER TABLE commercial_commission_ledger
    ADD COLUMN IF NOT EXISTS reconciled_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reconciled_by UUID REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS paid_by UUID REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS payment_reference VARCHAR(120);

CREATE TABLE commercial_commission_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    ledger_code BIGINT NOT NULL REFERENCES commercial_commission_ledger(code),
    event_type VARCHAR(30) NOT NULL CHECK (event_type IN ('CONCILIADA','PAGA')),
    before_state JSONB NOT NULL,
    after_state JSONB NOT NULL,
    reason TEXT NOT NULL,
    idempotency_key VARCHAR(160) NOT NULL,
    actor_id UUID NOT NULL REFERENCES users(id),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(enterprise_id, idempotency_key),
    FOREIGN KEY (enterprise_id, ledger_code)
        REFERENCES commercial_commission_ledger(enterprise_id, code)
);
CREATE INDEX idx_commercial_commission_events_tenant_ledger
    ON commercial_commission_events(enterprise_id, ledger_code, occurred_at DESC);
