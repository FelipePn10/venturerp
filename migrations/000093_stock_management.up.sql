BEGIN;
-- As migrations 014/015 criaram esquemas preliminares destas tabelas (com
-- product_id, sem item_code). Removemos esses esquemas antigos para recriar com
-- o schema correto deste módulo — mesmo padrão da migration 094 (production_orders).
DROP TABLE IF EXISTS public.physical_inventory_items CASCADE;
DROP TABLE IF EXISTS public.physical_inventories CASCADE;
DROP TABLE IF EXISTS public.stock_reservations CASCADE;
DROP TABLE IF EXISTS public.stock_balances CASCADE;
DROP TABLE IF EXISTS public.stock_movements CASCADE;

-- Stock movements (movimentacao de estoque)
CREATE TABLE IF NOT EXISTS public.stock_movements (
    id                      BIGSERIAL PRIMARY KEY,
    item_code               BIGINT NOT NULL,
    mask                    VARCHAR(200) NOT NULL DEFAULT '',
    warehouse_id            BIGINT NOT NULL,
    movement_type           VARCHAR(20) NOT NULL,  -- IN, OUT, TRANSFER_IN, TRANSFER_OUT, ADJUSTMENT, RESERVATION, UNRESERVATION
    quantity                NUMERIC(15,4) NOT NULL,
    unit_price              NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_price             NUMERIC(15,4) NOT NULL DEFAULT 0,
    reference_type          VARCHAR(30),        -- PURCHASE_ORDER, PRODUCTION_ORDER, SALES_ORDER, INVENTORY, MANUAL
    reference_code          BIGINT,
    lot                     VARCHAR(50),
    serial_number           VARCHAR(50),
    batch                   VARCHAR(50),
    expiration_date         DATE,
    notes                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL
);

-- Stock reservations
CREATE TABLE IF NOT EXISTS public.stock_reservations (
    id                      BIGSERIAL PRIMARY KEY,
    item_code               BIGINT NOT NULL,
    mask                    VARCHAR(200) NOT NULL DEFAULT '',
    warehouse_id            BIGINT NOT NULL,
    quantity                NUMERIC(15,4) NOT NULL,
    reference_type          VARCHAR(30) NOT NULL, -- SALES_ORDER, PRODUCTION_ORDER, MANUAL
    reference_code          BIGINT NOT NULL,
    reference_item_code     BIGINT,              -- sales_order_item or production_order id
    reservation_date        DATE NOT NULL DEFAULT CURRENT_DATE,
    expiration_date         DATE,
    status                  VARCHAR(20) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, CONSUMED, CANCELLED
    notes                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL
);

-- Stock balance (current snapshot - updated by movements)
CREATE TABLE IF NOT EXISTS public.stock_balances (
    id                      BIGSERIAL PRIMARY KEY,
    item_code               BIGINT NOT NULL,
    mask                    VARCHAR(200) NOT NULL DEFAULT '',
    warehouse_id            BIGINT NOT NULL,
    quantity                NUMERIC(15,4) NOT NULL DEFAULT 0,
    reserved_qty            NUMERIC(15,4) NOT NULL DEFAULT 0,
    available_qty           NUMERIC(15,4) GENERATED ALWAYS AS (quantity - reserved_qty) STORED,
    minimum_stock           NUMERIC(15,4) NOT NULL DEFAULT 0,
    maximum_stock           NUMERIC(15,4) NOT NULL DEFAULT 0,
    safety_stock            NUMERIC(15,4) NOT NULL DEFAULT 0,
    avg_cost                NUMERIC(15,4) NOT NULL DEFAULT 0,
    last_cost               NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_cost              NUMERIC(15,4) NOT NULL DEFAULT 0,
    last_movement_at        TIMESTAMPTZ,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(item_code, mask, warehouse_id)
);

-- Physical inventory
CREATE TABLE IF NOT EXISTS public.physical_inventories (
    id                      BIGSERIAL PRIMARY KEY,
    code                    BIGINT NOT NULL,
    description             VARCHAR(200) NOT NULL,
    warehouse_id            BIGINT NOT NULL,
    start_date              DATE NOT NULL,
    end_date                DATE,
    status                  VARCHAR(20) NOT NULL DEFAULT 'OPEN',  -- OPEN, IN_PROGRESS, COUNTED, ADJUSTED, CLOSED
    total_items             INT NOT NULL DEFAULT 0,
    counted_items           INT NOT NULL DEFAULT 0,
    notes                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL
);

-- Physical inventory items (counting)
CREATE TABLE IF NOT EXISTS public.physical_inventory_items (
    id                      BIGSERIAL PRIMARY KEY,
    inventory_id            BIGINT NOT NULL REFERENCES public.physical_inventories(id),
    item_code               BIGINT NOT NULL,
    mask                    VARCHAR(200) NOT NULL DEFAULT '',
    warehouse_id            BIGINT NOT NULL,
    system_qty              NUMERIC(15,4) NOT NULL DEFAULT 0,
    counted_qty             NUMERIC(15,4),
    difference_qty          NUMERIC(15,4),
    unit_cost               NUMERIC(15,4),
    adjustment_type         VARCHAR(20),  -- SURPLUS, SHORTAGE, NONE
    adjustment_reason       VARCHAR(100),
    counted_by              UUID,
    counted_at              TIMESTAMPTZ,
    is_adjusted             BOOLEAN NOT NULL DEFAULT FALSE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stock_movements_item ON public.stock_movements(item_code);
CREATE INDEX idx_stock_movements_warehouse ON public.stock_movements(warehouse_id);
CREATE INDEX idx_stock_movements_date ON public.stock_movements(created_at);
CREATE INDEX idx_stock_reservations_item ON public.stock_reservations(item_code);
CREATE INDEX idx_stock_reservations_status ON public.stock_reservations(status);
CREATE INDEX idx_stock_balances_item ON public.stock_balances(item_code);
CREATE INDEX idx_physical_inventory_status ON public.physical_inventories(status);
COMMIT;
