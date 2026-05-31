BEGIN;

-- ─── Cotação de Compra ─────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS purchase_quotations (
    id              BIGSERIAL PRIMARY KEY,
    code            BIGINT      NOT NULL UNIQUE,
    enterprise_code BIGINT      NOT NULL,
    status          VARCHAR(12) NOT NULL DEFAULT 'OPEN'
                    CHECK (status IN ('OPEN','QUOTED','CLOSED','CANCELLED')),
    emission_date   DATE        NOT NULL DEFAULT CURRENT_DATE,
    notes           TEXT,
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID        NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Itens a cotar — originados de solicitação, ordem planejada ou manual.
CREATE TABLE IF NOT EXISTS purchase_quotation_items (
    id             BIGSERIAL PRIMARY KEY,
    quotation_code BIGINT        NOT NULL REFERENCES purchase_quotations(code) ON DELETE CASCADE,
    sequence       INT           NOT NULL DEFAULT 1,
    item_code      BIGINT        NOT NULL,
    quantity       NUMERIC(15,4) NOT NULL DEFAULT 0,
    uom            VARCHAR(10),
    delivery_date  DATE,
    source_type    VARCHAR(15)   NOT NULL DEFAULT 'MANUAL'
                   CHECK (source_type IN ('REQUISITION','PLANNED_ORDER','MANUAL')),
    source_code    BIGINT,        -- código da solicitação / ordem planejada
    source_item_id BIGINT,        -- id do item da solicitação (p/ registrar atendimento)
    is_configured  BOOLEAN       NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- Fornecedores convidados a cotar.
CREATE TABLE IF NOT EXISTS purchase_quotation_suppliers (
    id             BIGSERIAL PRIMARY KEY,
    quotation_code BIGINT      NOT NULL REFERENCES purchase_quotations(code) ON DELETE CASCADE,
    supplier_code  BIGINT      NOT NULL,
    invited_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (quotation_code, supplier_code)
);

-- Respostas (preços) dos fornecedores por item cotado.
CREATE TABLE IF NOT EXISTS purchase_quotation_prices (
    id                BIGSERIAL PRIMARY KEY,
    quotation_item_id BIGINT        NOT NULL REFERENCES purchase_quotation_items(id) ON DELETE CASCADE,
    supplier_code     BIGINT        NOT NULL,
    unit_price        NUMERIC(18,6) NOT NULL DEFAULT 0,
    lead_time_days    INT           NOT NULL DEFAULT 0,
    payment_term_code BIGINT,
    notes             TEXT,
    is_selected       BOOLEAN       NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (quotation_item_id, supplier_code)
);

CREATE INDEX IF NOT EXISTS ix_quotation_items_quotation ON purchase_quotation_items(quotation_code);
CREATE INDEX IF NOT EXISTS ix_quotation_prices_item ON purchase_quotation_prices(quotation_item_id);

COMMIT;
