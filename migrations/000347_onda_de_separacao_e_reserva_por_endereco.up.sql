-- Separação por onda e reserva no nível do endereço/lote.
--
-- Sem reserva por endereço, duas ondas geradas no mesmo minuto sugerem o MESMO
-- lote no MESMO endereço: a primeira leva a peça, a segunda chega ao endereço e
-- não acha nada. A reserva existia só por item/almoxarifado, o que não resolve —
-- é justamente o endereço que o separador visita.
--
-- A onda agrupa várias necessidades numa caminhada só. É o ganho real de
-- produtividade da separação: sem ela, dez pedidos viram dez percursos pelo
-- galpão, cada um passando pelos mesmos corredores.

ALTER TABLE stock_reservations
    ADD COLUMN IF NOT EXISTS address VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS lot     VARCHAR(50)  NOT NULL DEFAULT '';

COMMENT ON COLUMN stock_reservations.address IS
    'Endereço reservado; vazio = reserva do almoxarifado, sem endereçamento.';

-- Reservado por lote/endereço: é o que o FEFO desconta para não prometer duas
-- vezes a mesma peça.
ALTER TABLE stock_lot_balances
    ADD COLUMN IF NOT EXISTS reserved_qty NUMERIC(18,6) NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_stock_reservations_endereco
    ON stock_reservations (enterprise_id, warehouse_id, address, status);

CREATE TABLE IF NOT EXISTS stock_picking_waves (
    id            BIGSERIAL   PRIMARY KEY,
    enterprise_id BIGINT      NOT NULL REFERENCES enterprise(id),
    code          BIGINT      NOT NULL,
    warehouse_id  BIGINT      NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'ABERTA',
    rule          VARCHAR(10) NOT NULL DEFAULT 'FEFO',
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by    UUID        NOT NULL,
    confirmed_at  TIMESTAMPTZ,
    cancelled_at  TIMESTAMPTZ,
    CONSTRAINT uq_picking_waves_code UNIQUE (enterprise_id, code),
    CONSTRAINT chk_picking_wave_status CHECK (status IN ('ABERTA','SEPARADA','CANCELADA'))
);

CREATE TABLE IF NOT EXISTS stock_picking_wave_lines (
    id             BIGSERIAL      PRIMARY KEY,
    enterprise_id  BIGINT         NOT NULL REFERENCES enterprise(id),
    wave_id        BIGINT         NOT NULL REFERENCES stock_picking_waves(id) ON DELETE CASCADE,
    item_code      BIGINT         NOT NULL,
    mask           VARCHAR(200)   NOT NULL DEFAULT '',
    lot            VARCHAR(50)    NOT NULL DEFAULT '',
    address        VARCHAR(100)   NOT NULL DEFAULT '',
    pick_sequence  INTEGER        NOT NULL DEFAULT 0,
    quantity       NUMERIC(18,6)  NOT NULL,
    picked_qty     NUMERIC(18,6)  NOT NULL DEFAULT 0,
    reservation_id BIGINT         REFERENCES stock_reservations(id),
    -- A necessidade que originou a linha, para a conferência saber a quem
    -- entregar o que foi separado.
    reference_type VARCHAR(30),
    reference_code BIGINT,
    CONSTRAINT chk_wave_line_qty CHECK (quantity > 0)
);

CREATE INDEX IF NOT EXISTS idx_wave_lines_onda  ON stock_picking_wave_lines (wave_id, pick_sequence, address);
CREATE INDEX IF NOT EXISTS idx_picking_waves_st ON stock_picking_waves (enterprise_id, status);
