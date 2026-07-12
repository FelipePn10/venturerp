ALTER TABLE production_orders
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);

UPDATE production_orders po
SET enterprise_id = planned.enterprise_id
FROM planned_orders planned
WHERE planned.code = po.planned_order_id AND po.enterprise_id IS NULL;

UPDATE production_orders
SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_production_orders_tenant ON production_orders (enterprise_id, id);

CREATE TABLE production_deliveries (
    id BIGSERIAL PRIMARY KEY,
    production_order_id BIGINT NOT NULL REFERENCES production_orders(id),
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    idempotency_key VARCHAR(100) NOT NULL,
    quantity NUMERIC(18,6) NOT NULL CHECK (quantity >= 0),
    movement_class VARCHAR(3) NOT NULL CHECK (movement_class IN ('EP','EPP','EPE')),
    warehouse_id BIGINT NOT NULL,
    lot VARCHAR(100),
    is_final BOOLEAN NOT NULL DEFAULT FALSE,
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    UNIQUE (enterprise_id, idempotency_key)
);

CREATE INDEX idx_production_deliveries_order
    ON production_deliveries (enterprise_id, production_order_id, delivered_at);

CREATE TABLE production_order_service_links (
    production_order_id BIGINT NOT NULL REFERENCES production_orders(id) ON DELETE CASCADE,
    purchase_order_code BIGINT NOT NULL,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    PRIMARY KEY (enterprise_id, production_order_id, purchase_order_code)
);
