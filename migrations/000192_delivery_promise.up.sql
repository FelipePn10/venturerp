CREATE TABLE IF NOT EXISTS delivery_tank_reservation_sequences (
    id BIGINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    last_code BIGINT NOT NULL DEFAULT 0
);

INSERT INTO delivery_tank_reservation_sequences (id, last_code)
VALUES (1, 0)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS delivery_tank_reservations (
    id BIGSERIAL PRIMARY KEY,
    code BIGINT NOT NULL UNIQUE,
    customer_code BIGINT,
    item_code BIGINT NOT NULL,
    mask VARCHAR(200) NOT NULL DEFAULT '',
    tank_code BIGINT NOT NULL DEFAULT 0,
    requested_qty NUMERIC(15,4) NOT NULL,
    reserved_qty NUMERIC(15,4) NOT NULL,
    allocation_date DATE NOT NULL,
    expires_at DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    CONSTRAINT delivery_tank_reservations_status_chk
        CHECK (status IN ('ACTIVE', 'CANCELLED', 'EXPIRED'))
);

CREATE INDEX IF NOT EXISTS idx_delivery_tank_reservations_active
    ON delivery_tank_reservations (tank_code, allocation_date)
    WHERE status = 'ACTIVE';

CREATE INDEX IF NOT EXISTS idx_delivery_tank_reservations_item
    ON delivery_tank_reservations (item_code, mask);
