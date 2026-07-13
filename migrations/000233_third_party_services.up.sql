BEGIN;

CREATE TABLE third_party_service_prices (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    item_code BIGINT NOT NULL,
    mask VARCHAR(255) NOT NULL DEFAULT '',
    supplier_code BIGINT NOT NULL,
    operation_id BIGINT NOT NULL REFERENCES operations(id),
    uom VARCHAR(10) NOT NULL,
    reference_date DATE NOT NULL,
    preferred BOOLEAN NOT NULL DEFAULT FALSE,
    unit_price NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (unit_price >= 0),
    conversion_factor NUMERIC(18,8) CHECK (conversion_factor IS NULL OR conversion_factor > 0),
    freight_type VARCHAR(10) NOT NULL DEFAULT 'FIXED' CHECK (freight_type IN ('FIXED','PERCENT')),
    freight_value NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (freight_value >= 0),
    tax_percent NUMERIC(9,6) NOT NULL DEFAULT 0 CHECK (tax_percent BETWEEN 0 AND 100),
    formula TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id,item_code,mask,supplier_code,operation_id,reference_date)
);
CREATE UNIQUE INDEX uq_third_party_service_preferred
    ON third_party_service_prices(enterprise_id,item_code,mask,operation_id,reference_date)
    WHERE preferred AND is_active;
CREATE INDEX idx_third_party_service_price_lookup
    ON third_party_service_prices(enterprise_id,item_code,mask,operation_id,supplier_code,reference_date DESC);

CREATE TABLE third_party_service_price_rules (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    price_id BIGINT NOT NULL REFERENCES third_party_service_prices(id) ON DELETE CASCADE,
    characteristic VARCHAR(100) NOT NULL,
    answer VARCHAR(200),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(price_id,characteristic,answer)
);

CREATE TABLE third_party_service_price_history (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    price_id BIGINT NOT NULL,
    action VARCHAR(20) NOT NULL CHECK(action IN ('CREATE','UPDATE','READJUST','COPY','MOVE','DELETE')),
    reason VARCHAR(300) NOT NULL,
    snapshot JSONB NOT NULL,
    changed_by UUID NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_third_party_service_price_history ON third_party_service_price_history(enterprise_id,price_id,changed_at DESC);

CREATE TABLE third_party_service_orders (
    id BIGSERIAL PRIMARY KEY,
    code BIGINT NOT NULL,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    production_order_id BIGINT NOT NULL REFERENCES production_orders(id) ON DELETE CASCADE,
    route_operation_id BIGINT NOT NULL REFERENCES route_operations(id),
    operation_id BIGINT NOT NULL REFERENCES operations(id),
    item_code BIGINT NOT NULL,
    mask VARCHAR(255) NOT NULL DEFAULT '',
    supplier_code BIGINT,
    service_item_code BIGINT,
    uom VARCHAR(10) NOT NULL,
    quantity NUMERIC(18,6) NOT NULL CHECK(quantity > 0),
    fulfilled_quantity NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK(fulfilled_quantity >= 0),
    start_date DATE NOT NULL,
    due_date DATE NOT NULL,
    status VARCHAR(30) NOT NULL CHECK(status IN ('PLANNED','FIRM','RELEASED_WITH_PO','RELEASED_WITHOUT_PO','COMPLETED','CANCELLED')),
    purchase_requisition_code BIGINT,
    purchase_order_code BIGINT,
    remittance_type VARCHAR(20) NOT NULL DEFAULT 'DEMAND_ITEMS' CHECK(remittance_type IN ('DEMAND_ITEMS','ORDER_ITEM','GENERIC','NONE')),
    kanban BOOLEAN NOT NULL DEFAULT FALSE,
    notes TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(enterprise_id,code), UNIQUE(enterprise_id,production_order_id,route_operation_id),
    CHECK(fulfilled_quantity <= quantity), CHECK(due_date >= start_date)
);
CREATE INDEX idx_third_party_service_orders_query ON third_party_service_orders(enterprise_id,status,due_date,supplier_code,item_code);

CREATE TABLE third_party_service_movements (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    service_order_id BIGINT NOT NULL REFERENCES third_party_service_orders(id) ON DELETE CASCADE,
    movement_type VARCHAR(20) NOT NULL CHECK(movement_type IN ('REMITTANCE','RETURN','RECEIPT','ADJUSTMENT')),
    quantity NUMERIC(18,6) NOT NULL CHECK(quantity > 0),
    occurred_at TIMESTAMPTZ NOT NULL,
    reference_type VARCHAR(30), reference_code VARCHAR(100), notes TEXT,
    created_by UUID NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_third_party_service_movements_order ON third_party_service_movements(enterprise_id,service_order_id,occurred_at);
COMMIT;
