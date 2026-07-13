BEGIN;

CREATE TABLE production_resource_groups (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    code VARCHAR(30) NOT NULL,
    description VARCHAR(200) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, code)
);

CREATE TABLE machine_calendars (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    code BIGINT NOT NULL,
    description VARCHAR(200) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, code)
);

CREATE TABLE machine_calendar_intervals (
    id BIGSERIAL PRIMARY KEY,
    calendar_id BIGINT NOT NULL REFERENCES machine_calendars(id) ON DELETE CASCADE,
    weekday SMALLINT NOT NULL CHECK (weekday BETWEEN 0 AND 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    CHECK (end_time > start_time),
    UNIQUE (calendar_id, weekday, start_time, end_time)
);

ALTER TABLE machine_types
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id),
    ADD COLUMN IF NOT EXISTS machine_cost_center_id BIGINT REFERENCES cost_centers(id),
    ADD COLUMN IF NOT EXISTS labor_cost_center_id BIGINT REFERENCES cost_centers(id),
    ADD COLUMN IF NOT EXISTS capacity_hours NUMERIC(12,4) NOT NULL DEFAULT 8,
    ADD CONSTRAINT machine_types_distinct_cost_centers CHECK (
        labor_cost_center_id IS NULL OR labor_cost_center_id <> machine_cost_center_id
    );

ALTER TABLE machines
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id),
    ADD COLUMN IF NOT EXISTS resource_group_id BIGINT REFERENCES production_resource_groups(id),
    ADD COLUMN IF NOT EXISTS calendar_id BIGINT REFERENCES machine_calendars(id),
    ADD COLUMN IF NOT EXISTS location VARCHAR(200),
    ADD COLUMN IF NOT EXISTS is_critical BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE production_order_operations
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE production_sequences
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id),
    ADD COLUMN IF NOT EXISTS machine_id BIGINT REFERENCES machines(id);

UPDATE production_order_operations operation
SET enterprise_id = production_order.enterprise_id
FROM production_orders production_order
WHERE operation.production_order_id = production_order.id AND operation.enterprise_id IS NULL;
UPDATE production_sequences sequence
SET enterprise_id = production_order.enterprise_id
FROM production_orders production_order
WHERE sequence.production_order_id = production_order.id AND sequence.enterprise_id IS NULL;

ALTER TABLE production_order_operations ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE production_sequences ALTER COLUMN enterprise_id SET NOT NULL;

CREATE OR REPLACE FUNCTION set_production_sequence_enterprise() RETURNS trigger AS $$
BEGIN
    SELECT enterprise_id INTO NEW.enterprise_id FROM production_orders WHERE id = NEW.production_order_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_production_sequence_enterprise BEFORE INSERT OR UPDATE OF production_order_id
ON production_sequences FOR EACH ROW EXECUTE FUNCTION set_production_sequence_enterprise();

CREATE OR REPLACE FUNCTION set_production_operation_enterprise() RETURNS trigger AS $$
BEGIN
    SELECT enterprise_id INTO NEW.enterprise_id FROM production_orders WHERE id = NEW.production_order_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_production_operation_enterprise BEFORE INSERT OR UPDATE OF production_order_id
ON production_order_operations FOR EACH ROW EXECUTE FUNCTION set_production_operation_enterprise();

CREATE TABLE manufacturing_sequencing_settings (
    enterprise_id BIGINT PRIMARY KEY REFERENCES enterprise(id) ON DELETE CASCADE,
    list_only_active_resources BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_resource_groups_enterprise ON production_resource_groups(enterprise_id);
CREATE INDEX idx_machine_calendars_enterprise ON machine_calendars(enterprise_id);
CREATE INDEX idx_machines_sequencing ON machines(enterprise_id, resource_group_id, machine_type_code, is_active);
CREATE INDEX idx_production_sequences_tenant_range ON production_sequences(enterprise_id, scheduled_start, scheduled_end);
CREATE INDEX idx_production_operations_tenant ON production_order_operations(enterprise_id, production_order_id);

COMMIT;
