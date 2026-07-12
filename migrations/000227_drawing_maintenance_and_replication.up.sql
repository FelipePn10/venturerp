CREATE TABLE manufacturing_item_parameters (
    enterprise_id BIGINT PRIMARY KEY REFERENCES enterprise(id) ON DELETE CASCADE,
    parameter_8_replicate_drawing_revision BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID NOT NULL REFERENCES users(id)
);

CREATE TABLE item_engineering_drawings (
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    item_code BIGINT NOT NULL,
    mask VARCHAR(200) NOT NULL DEFAULT '',
    drawing_code VARCHAR(120) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID NOT NULL REFERENCES users(id),
    PRIMARY KEY (enterprise_id, item_code, mask)
);

CREATE INDEX idx_item_engineering_drawings_code
    ON item_engineering_drawings (enterprise_id, drawing_code);
