BEGIN;

-- ─── Tabela de Preço de Compra ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS purchase_price_tables (
    id             BIGSERIAL PRIMARY KEY,
    code           BIGINT       NOT NULL UNIQUE,
    description    VARCHAR(150) NOT NULL,
    currency_code  VARCHAR(5)   NOT NULL DEFAULT 'BRL',
    validity_start DATE,
    validity_end   DATE,
    is_active      BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by     UUID         NOT NULL,
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Itens da tabela. supplier_code opcional permite preço específico por fornecedor;
-- quando NULL, o preço vale para qualquer fornecedor.
CREATE TABLE IF NOT EXISTS purchase_price_table_items (
    id            BIGSERIAL PRIMARY KEY,
    table_id      BIGINT        NOT NULL REFERENCES purchase_price_tables(id) ON DELETE CASCADE,
    item_code     BIGINT        NOT NULL,
    supplier_code BIGINT,
    uom           VARCHAR(10),                                  -- UM da tabela (1º na hierarquia)
    price         NUMERIC(18,6) NOT NULL DEFAULT 0,
    min_qty       NUMERIC(18,4) NOT NULL DEFAULT 0,
    is_active     BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_purchase_price_item
    ON purchase_price_table_items(table_id, item_code, COALESCE(supplier_code, 0));
CREATE INDEX IF NOT EXISTS ix_purchase_price_item_lookup
    ON purchase_price_table_items(table_id, item_code) WHERE is_active = TRUE;

-- Liga o default do fornecedor (pasta Empresas) à tabela de preço de compra.
ALTER TABLE supplier_enterprises
    ADD CONSTRAINT fk_supplier_enterprises_price_table
    FOREIGN KEY (purchase_price_table_id) REFERENCES purchase_price_tables(id);

COMMIT;
