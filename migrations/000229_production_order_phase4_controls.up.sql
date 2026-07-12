BEGIN;

ALTER TABLE production_orders
    ADD COLUMN temporary_lot_code VARCHAR(100),
    ADD COLUMN temporary_lot_manufactured_on DATE,
    ADD COLUMN temporary_lot_expires_on DATE;

ALTER TABLE production_orders ADD CONSTRAINT production_order_temp_lot_dates_chk
    CHECK (temporary_lot_expires_on IS NULL OR
           (temporary_lot_manufactured_on IS NOT NULL AND temporary_lot_expires_on >= temporary_lot_manufactured_on));

ALTER TABLE production_order_materials
    ADD COLUMN uom VARCHAR(20) NOT NULL DEFAULT 'UN',
    ADD COLUMN controls_lot BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN controls_address BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE manufacturing_stock_item_controls (
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    item_code BIGINT NOT NULL,
    stock_uom VARCHAR(20) NOT NULL DEFAULT 'UN',
    controls_lot BOOLEAN NOT NULL DEFAULT FALSE,
    controls_address BOOLEAN NOT NULL DEFAULT FALSE,
    inventory_group_type VARCHAR(20) NOT NULL DEFAULT 'STANDARD'
        CHECK (inventory_group_type IN ('STANDARD','SECONDARY_MATERIAL')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_id,item_code)
);

CREATE TABLE manufacturing_stock_parameters (
    enterprise_id BIGINT PRIMARY KEY REFERENCES enterprise(id),
    lot_return_mode CHAR(1) NOT NULL DEFAULT 'I' CHECK (lot_return_mode IN ('A','I','E')),
    auto_issue_lots BOOLEAN NOT NULL DEFAULT FALSE,
    movement_from DATE,
    movement_to DATE,
    CHECK (movement_from IS NULL OR movement_to IS NULL OR movement_from <= movement_to)
);

CREATE TABLE manufacturing_stock_closed_periods (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    period_from DATE NOT NULL,
    period_to DATE NOT NULL,
    CHECK (period_from <= period_to),
    UNIQUE (enterprise_id,period_from,period_to)
);

CREATE TABLE manufacturing_warehouse_addresses (
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    warehouse_id BIGINT NOT NULL,
    address VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (enterprise_id,warehouse_id,address)
);

ALTER TABLE production_order_scrap_destinations
    ADD COLUMN destination_kind VARCHAR(10) NOT NULL DEFAULT 'ORDER_ITEM'
        CHECK (destination_kind IN ('ORDER_ITEM','DEMAND')),
    ADD COLUMN return_quantity NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (return_quantity >= 0),
    ADD COLUMN scrap_quantity NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (scrap_quantity >= 0),
    ADD COLUMN source_uom VARCHAR(20) NOT NULL DEFAULT 'UN',
    ADD COLUMN scrap_uom VARCHAR(20) NOT NULL DEFAULT 'UN';

UPDATE production_order_scrap_destinations SET scrap_quantity=quantity;

CREATE INDEX idx_manufacturing_stock_closed_periods
    ON manufacturing_stock_closed_periods(enterprise_id,period_from,period_to);
CREATE INDEX idx_manufacturing_warehouse_addresses
    ON manufacturing_warehouse_addresses(enterprise_id,warehouse_id) WHERE is_active;
CREATE INDEX idx_production_order_scrap_scope
    ON production_order_scrap_destinations(enterprise_id,production_order_id,destination_kind,destination_date);

COMMIT;
