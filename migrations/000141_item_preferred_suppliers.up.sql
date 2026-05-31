BEGIN;

-- ─── Fornecedor preferencial por item / Descrição de itens por fornecedor ──────
-- Liga um item a fornecedores com ranking de preferência (1 = preferencial).
-- Também guarda código/descrição/UM do item no fornecedor (2º nível da hierarquia
-- de UM/descrição do Pedido de Compra).
CREATE TABLE IF NOT EXISTS item_preferred_suppliers (
    id                   BIGSERIAL PRIMARY KEY,
    item_code            BIGINT       NOT NULL,
    supplier_code        BIGINT       NOT NULL,
    ranking              INT          NOT NULL DEFAULT 1,
    supplier_item_code   VARCHAR(60),                 -- código do item no fornecedor
    supplier_description VARCHAR(200),                -- descrição cfe. fornecedor
    uom                  VARCHAR(10),                 -- UM do item no fornecedor
    lead_time_days       INT          NOT NULL DEFAULT 0,
    is_active            BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by           UUID         NOT NULL,
    UNIQUE (item_code, supplier_code)
);

CREATE INDEX IF NOT EXISTS ix_item_preferred_suppliers_item
    ON item_preferred_suppliers(item_code) WHERE is_active = TRUE;

COMMIT;
