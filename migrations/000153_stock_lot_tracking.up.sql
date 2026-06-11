BEGIN;

-- Registro de lote/corrida: identifica a matéria-prima por lote do fornecedor,
-- número da corrida (heat number) e certificado de qualidade do material — base
-- da rastreabilidade exigida por clientes de metalurgia.
CREATE TABLE IF NOT EXISTS public.stock_lots (
    id              BIGSERIAL PRIMARY KEY,
    item_code       BIGINT NOT NULL,
    lot             VARCHAR(50) NOT NULL,
    heat_number     VARCHAR(50),          -- corrida
    certificate     VARCHAR(120),         -- certificado de qualidade do material
    supplier_code   BIGINT,
    received_at     DATE,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID NOT NULL,
    UNIQUE (item_code, lot)
);

CREATE INDEX IF NOT EXISTS idx_stock_lots_item ON public.stock_lots(item_code);

-- Saldo segregado por lote: ao contrário de stock_balances (por item/depósito), o
-- saldo de lote permite saber quanto de cada corrida ainda existe em cada depósito.
CREATE TABLE IF NOT EXISTS public.stock_lot_balances (
    id                BIGSERIAL PRIMARY KEY,
    item_code         BIGINT NOT NULL,
    mask              VARCHAR(200) NOT NULL DEFAULT '',
    warehouse_id      BIGINT NOT NULL,
    lot               VARCHAR(50) NOT NULL,
    quantity          NUMERIC(15,4) NOT NULL DEFAULT 0,
    last_cost         NUMERIC(15,4) NOT NULL DEFAULT 0,
    last_movement_at  TIMESTAMPTZ,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (item_code, mask, warehouse_id, lot)
);

CREATE INDEX IF NOT EXISTS idx_stock_lot_balances_item ON public.stock_lot_balances(item_code);
CREATE INDEX IF NOT EXISTS idx_stock_lot_balances_lot ON public.stock_lot_balances(item_code, lot);

COMMIT;
