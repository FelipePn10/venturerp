BEGIN;

-- Expedição / Carregamento (romaneio): separação, conferência e despacho de
-- pedidos de venda. Logística de saída, distinta da baixa fiscal (NF-e saída).
CREATE TABLE IF NOT EXISTS shipments (
    id                BIGSERIAL PRIMARY KEY,
    code              BIGINT       NOT NULL UNIQUE,
    sales_order_code  BIGINT,
    carrier_code      BIGINT,
    status            VARCHAR(20)  NOT NULL DEFAULT 'OPEN', -- OPEN, SEPARATED, CONFERRED, SHIPPED, CANCELLED
    total_volumes     INT          NOT NULL DEFAULT 0,
    total_weight      NUMERIC(15,4) NOT NULL DEFAULT 0,
    notes             TEXT,
    shipped_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by        UUID         NOT NULL
);

CREATE TABLE IF NOT EXISTS shipment_items (
    id                    BIGSERIAL PRIMARY KEY,
    shipment_id           BIGINT       NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    sequence              INT          NOT NULL DEFAULT 0,
    item_code             BIGINT       NOT NULL,
    sales_order_item_code BIGINT,
    warehouse_id          BIGINT,
    quantity              NUMERIC(15,4) NOT NULL DEFAULT 0,
    conferred_qty         NUMERIC(15,4) NOT NULL DEFAULT 0,
    is_conferred          BOOLEAN      NOT NULL DEFAULT FALSE,
    notes                 TEXT,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS shipment_sequences (
    id          INT PRIMARY KEY DEFAULT 1,
    last_number BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_shipments_sales_order ON shipments(sales_order_code);
CREATE INDEX IF NOT EXISTS idx_shipment_items_shipment ON shipment_items(shipment_id);

COMMIT;
