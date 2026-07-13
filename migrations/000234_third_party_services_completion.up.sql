BEGIN;

CREATE TABLE global_unit_conversions (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    from_uom VARCHAR(10) NOT NULL,
    to_uom VARCHAR(10) NOT NULL,
    factor NUMERIC(18,8) NOT NULL CHECK(factor>0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK(from_uom<>to_uom),
    UNIQUE(enterprise_id,from_uom,to_uom)
);

ALTER TABLE item_unit_conversions
    ADD COLUMN mask VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN rounding_percent NUMERIC(9,6) NOT NULL DEFAULT 0 CHECK(rounding_percent BETWEEN 0 AND 100),
    ADD COLUMN tolerance_value NUMERIC(18,8) NOT NULL DEFAULT 0 CHECK(tolerance_value>=0),
    ADD COLUMN tolerance_type VARCHAR(10) NOT NULL DEFAULT 'VALUE' CHECK(tolerance_type IN ('VALUE','PERCENT'));
ALTER TABLE item_unit_conversions DROP CONSTRAINT IF EXISTS item_unit_conversions_item_code_from_uom_to_uom_key;
CREATE UNIQUE INDEX uq_item_unit_conversions_config
    ON item_unit_conversions(item_code,mask,from_uom,to_uom);

ALTER TABLE third_party_service_movements
    ADD COLUMN idempotency_key VARCHAR(100),
    ADD COLUMN warehouse_id BIGINT,
    ADD COLUMN lot VARCHAR(100);
CREATE UNIQUE INDEX uq_third_party_service_movement_idempotency
    ON third_party_service_movements(enterprise_id,idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE TABLE third_party_service_order_history (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    service_order_id BIGINT NOT NULL REFERENCES third_party_service_orders(id) ON DELETE CASCADE,
    event_type VARCHAR(30) NOT NULL,
    previous_status VARCHAR(30),
    new_status VARCHAR(30),
    quantity NUMERIC(18,6),
    reference_type VARCHAR(30),
    reference_code VARCHAR(100),
    actor_id UUID NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_third_party_service_order_history
    ON third_party_service_order_history(enterprise_id,service_order_id,occurred_at);

COMMIT;
