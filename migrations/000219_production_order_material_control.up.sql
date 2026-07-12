CREATE TABLE production_order_materials (
    id BIGSERIAL PRIMARY KEY,
    production_order_id BIGINT NOT NULL REFERENCES production_orders(id) ON DELETE CASCADE,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    material_kind VARCHAR(10) NOT NULL CHECK (material_kind IN ('DEMAND','RETURN')),
    item_code BIGINT NOT NULL,
    mask VARCHAR(200) NOT NULL DEFAULT '',
    substituted_item_code BIGINT,
    quantity NUMERIC(18,6) NOT NULL CHECK (quantity >= 0),
    attended_quantity NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (attended_quantity >= 0),
    warehouse_id BIGINT NOT NULL,
    automatic_issue BOOLEAN NOT NULL DEFAULT FALSE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    UNIQUE (enterprise_id, production_order_id, material_kind, item_code, mask, substituted_item_code)
);

CREATE INDEX idx_production_order_materials_order
    ON production_order_materials (enterprise_id, production_order_id, material_kind);

CREATE TABLE production_order_wms_requests (
    id BIGSERIAL PRIMARY KEY,
    production_order_material_id BIGINT NOT NULL REFERENCES production_order_materials(id) ON DELETE CASCADE,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    status VARCHAR(12) NOT NULL CHECK (status IN ('PENDING','SENT','SEPARATED','CANCELLED')),
    external_reference VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, production_order_material_id, external_reference)
);

CREATE TABLE warehouse_wms_settings (
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    warehouse_id BIGINT NOT NULL,
    is_wms BOOLEAN NOT NULL DEFAULT FALSE,
    intermediate_out_warehouse_id BIGINT,
    PRIMARY KEY (enterprise_id, warehouse_id),
    CHECK (intermediate_out_warehouse_id IS NULL OR intermediate_out_warehouse_id <> warehouse_id)
);

CREATE TABLE production_order_lot_allocations (
    id BIGSERIAL PRIMARY KEY,
    production_order_material_id BIGINT NOT NULL REFERENCES production_order_materials(id) ON DELETE CASCADE,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    movement_kind VARCHAR(11) NOT NULL CHECK (movement_kind IN ('REQUISITION','RETURN')),
    warehouse_id BIGINT NOT NULL,
    lot VARCHAR(100) NOT NULL,
    address VARCHAR(100),
    quantity NUMERIC(18,6) NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    UNIQUE (enterprise_id, production_order_material_id, movement_kind, warehouse_id, lot, address)
);

CREATE TABLE production_order_scrap_destinations (
    id BIGSERIAL PRIMARY KEY,
    production_order_id BIGINT NOT NULL REFERENCES production_orders(id) ON DELETE CASCADE,
    production_order_material_id BIGINT REFERENCES production_order_materials(id) ON DELETE CASCADE,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    scrap_item_code BIGINT NOT NULL,
    warehouse_id BIGINT NOT NULL,
    lot VARCHAR(100),
    address VARCHAR(100),
    quantity NUMERIC(18,6) NOT NULL CHECK (quantity > 0),
    destination_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL
);
