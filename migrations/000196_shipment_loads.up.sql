CREATE TABLE IF NOT EXISTS public.shipment_load_sequences (
    id BIGINT PRIMARY KEY,
    last_number BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS public.shipment_loads (
    id BIGSERIAL PRIMARY KEY,
    code BIGINT NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'PLANNED',
    description TEXT,
    carrier_code BIGINT,
    vehicle_plate VARCHAR(20),
    driver_name VARCHAR(120),
    driver_document VARCHAR(40),
    route_code VARCHAR(40),
    origin VARCHAR(120),
    destination VARCHAR(120),
    dispatch_box_code VARCHAR(40),
    planned_ship_date DATE,
    estimated_delivery DATE,
    started_loading_at TIMESTAMPTZ,
    loaded_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    shipped_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    total_shipments INTEGER NOT NULL DEFAULT 0,
    total_fiscal_notes INTEGER NOT NULL DEFAULT 0,
    total_volumes INTEGER NOT NULL DEFAULT 0,
    total_net_weight NUMERIC(18,4) NOT NULL DEFAULT 0,
    total_gross_weight NUMERIC(18,4) NOT NULL DEFAULT 0,
    total_cubage_m3 NUMERIC(18,6) NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    updated_by UUID
);

CREATE INDEX IF NOT EXISTS idx_shipment_loads_status ON public.shipment_loads(status);
CREATE INDEX IF NOT EXISTS idx_shipment_loads_carrier ON public.shipment_loads(carrier_code);
CREATE INDEX IF NOT EXISTS idx_shipment_loads_box ON public.shipment_loads(dispatch_box_code);
CREATE INDEX IF NOT EXISTS idx_shipment_loads_planned_date ON public.shipment_loads(planned_ship_date);

CREATE TABLE IF NOT EXISTS public.shipment_load_shipments (
    id BIGSERIAL PRIMARY KEY,
    load_id BIGINT NOT NULL REFERENCES public.shipment_loads(id) ON DELETE CASCADE,
    shipment_id BIGINT NOT NULL REFERENCES public.shipments(id) ON DELETE RESTRICT,
    sequence INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(load_id, shipment_id)
);

CREATE INDEX IF NOT EXISTS idx_shipment_load_shipments_shipment ON public.shipment_load_shipments(shipment_id);

CREATE TABLE IF NOT EXISTS public.shipment_load_fiscal_notes (
    id BIGSERIAL PRIMARY KEY,
    load_id BIGINT NOT NULL REFERENCES public.shipment_loads(id) ON DELETE CASCADE,
    shipment_id BIGINT REFERENCES public.shipments(id) ON DELETE SET NULL,
    fiscal_exit_id BIGINT NOT NULL,
    nfe_number BIGINT,
    nfe_key VARCHAR(80),
    sequence INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(load_id, fiscal_exit_id)
);

CREATE INDEX IF NOT EXISTS idx_shipment_load_fiscal_notes_shipment ON public.shipment_load_fiscal_notes(shipment_id);

CREATE TABLE IF NOT EXISTS public.shipment_delivery_instructions (
    id BIGSERIAL PRIMARY KEY,
    load_id BIGINT REFERENCES public.shipment_loads(id) ON DELETE CASCADE,
    customer_id BIGINT,
    title VARCHAR(120) NOT NULL,
    instruction TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 5,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shipment_delivery_instructions_load ON public.shipment_delivery_instructions(load_id);
CREATE INDEX IF NOT EXISTS idx_shipment_delivery_instructions_customer ON public.shipment_delivery_instructions(customer_id);

CREATE TABLE IF NOT EXISTS public.shipment_dispatch_boxes (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(40) NOT NULL UNIQUE,
    description TEXT,
    warehouse_id BIGINT,
    zone VARCHAR(80),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    current_load BIGINT REFERENCES public.shipment_loads(code) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shipment_dispatch_boxes_active ON public.shipment_dispatch_boxes(active);
