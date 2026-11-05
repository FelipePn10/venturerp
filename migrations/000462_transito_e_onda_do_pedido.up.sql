-- Fase E: transferência em duas etapas e onda de separação ligada ao pedido.

-- O material em trânsito fica num almoxarifado real de localização TRANSITO, e
-- segue no custo, na posição e no fechamento. Aqui fica de onde veio e para onde vai.
CREATE TABLE stock_transit_shipments (
    id                   BIGSERIAL     PRIMARY KEY,
    enterprise_id        BIGINT        NOT NULL REFERENCES enterprise(id),
    code                 BIGINT        NOT NULL,
    item_code            BIGINT        NOT NULL,
    mask                 VARCHAR(200)  NOT NULL DEFAULT '',
    lot                  VARCHAR(50),
    from_warehouse_id    BIGINT        NOT NULL REFERENCES warehouse(id),
    transit_warehouse_id BIGINT        NOT NULL REFERENCES warehouse(id),
    to_warehouse_id      BIGINT        NOT NULL REFERENCES warehouse(id),
    quantity             NUMERIC(20,6) NOT NULL CHECK (quantity > 0),
    received_qty         NUMERIC(20,6) NOT NULL DEFAULT 0 CHECK (received_qty >= 0),
    returned_qty         NUMERIC(20,6) NOT NULL DEFAULT 0 CHECK (returned_qty >= 0),
    lost_qty             NUMERIC(20,6) NOT NULL DEFAULT 0 CHECK (lost_qty >= 0),
    status               VARCHAR(12)   NOT NULL DEFAULT 'EM_TRANSITO' CHECK (status IN ('EM_TRANSITO', 'ENCERRADA')),
    expected_date        DATE,
    requisition_id       BIGINT        REFERENCES material_requisitions(id),
    notes                TEXT,
    created_by           UUID          NOT NULL,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    closed_at            TIMESTAMPTZ,
    UNIQUE (enterprise_id, code),
    CONSTRAINT ck_transito_saldo CHECK (received_qty + returned_qty + lost_qty <= quantity),
    CONSTRAINT ck_transito_destino CHECK (from_warehouse_id <> to_warehouse_id)
);
CREATE INDEX ix_stock_transit_abertos ON stock_transit_shipments (enterprise_id, item_code, to_warehouse_id) WHERE status = 'EM_TRANSITO';

-- O que a onda tirou da reserva do pedido, para devolver ao encerrar.
ALTER TABLE stock_picking_wave_lines
    ADD COLUMN order_reserved_qty NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (order_reserved_qty >= 0);

SELECT instalar_conferencia_de_empresa();
