BEGIN;
-- Migration 015 created production_orders with an old schema (no item_code).
-- Drop it so we can recreate with the correct schema.
DROP TABLE IF EXISTS public.production_orders CASCADE;

-- Production Order extends planned orders when they are firmed for production
CREATE TABLE IF NOT EXISTS public.production_orders (
    id                      BIGSERIAL PRIMARY KEY,
    order_number            BIGINT NOT NULL,
    planned_order_id        BIGINT REFERENCES public.planned_orders(id),
    item_code               BIGINT NOT NULL,
    mask                    VARCHAR(200) NOT NULL DEFAULT '',
    planned_qty             NUMERIC(15,4) NOT NULL,
    produced_qty            NUMERIC(15,4) NOT NULL DEFAULT 0,
    scrapped_qty            NUMERIC(15,4) NOT NULL DEFAULT 0,
    status                  VARCHAR(20) NOT NULL DEFAULT 'OPEN', -- OPEN, IN_PROGRESS, COMPLETED, CLOSED, CANCELLED
    start_date              DATE,
    end_date                DATE,
    machine_id              BIGINT REFERENCES public.machines(id),
    cost_center_id          BIGINT REFERENCES public.cost_centers(id),
    employee_id             BIGINT REFERENCES public.employees(id),
    priority                VARCHAR(50),
    notes                   TEXT,
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL
);

-- Production apportionment (apontamento de producao) - tracks time and quantity
CREATE TABLE IF NOT EXISTS public.production_appointments (
    id                      BIGSERIAL PRIMARY KEY,
    production_order_id     BIGINT NOT NULL REFERENCES public.production_orders(id),
    machine_id              BIGINT REFERENCES public.machines(id),
    employee_id             BIGINT REFERENCES public.employees(id),
    appointment_date        DATE NOT NULL DEFAULT CURRENT_DATE,
    start_time              TIME,
    end_time                TIME,
    produced_qty            NUMERIC(15,4) NOT NULL DEFAULT 0,
    scrapped_qty            NUMERIC(15,4) NOT NULL DEFAULT 0,
    scrap_reason            VARCHAR(100),
    notes                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL
);

-- Material consumption (baixa de materia-prima) - records what was consumed
CREATE TABLE IF NOT EXISTS public.production_consumptions (
    id                      BIGSERIAL PRIMARY KEY,
    production_order_id     BIGINT NOT NULL REFERENCES public.production_orders(id),
    appointment_id          BIGINT REFERENCES public.production_appointments(id),
    item_code               BIGINT NOT NULL,
    consumed_qty            NUMERIC(15,4) NOT NULL,
    warehouse_id            BIGINT,
    lot                     VARCHAR(50),
    consumption_date        DATE NOT NULL DEFAULT CURRENT_DATE,
    notes                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL
);

CREATE INDEX idx_po_order_item ON public.production_orders(item_code);
CREATE INDEX idx_po_order_status ON public.production_orders(status);
CREATE INDEX idx_po_appointment_order ON public.production_appointments(production_order_id);
CREATE INDEX idx_po_consumption_order ON public.production_consumptions(production_order_id);
COMMIT;
