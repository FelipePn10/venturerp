BEGIN;

-- Recreate the legacy thin BOM tables (retired by the up migration).
CREATE TABLE IF NOT EXISTS boms (
    id         BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id),
    bom_type   VARCHAR(10) NOT NULL CHECK (bom_type IN ('EBOM', 'MBOM')),
    version    INTEGER NOT NULL,
    status     VARCHAR(20) NOT NULL CHECK (status IN ('draft', 'released', 'obsolete')),
    valid_from DATE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    mask       BIGINT NOT NULL REFERENCES item_masks(id),
    UNIQUE (product_id, bom_type, version)
);

CREATE TABLE IF NOT EXISTS bom_items (
    id             BIGSERIAL PRIMARY KEY,
    bom_id         BIGINT NOT NULL REFERENCES boms(id),
    component_id   BIGINT NOT NULL REFERENCES products(id),
    quantity       NUMERIC(14,6) NOT NULL,
    uom            VARCHAR(10),
    scrap_percent  NUMERIC(5,2) NOT NULL DEFAULT 0,
    operation_id   BIGINT NOT NULL,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
    mask_component BIGINT NOT NULL REFERENCES component_masks(id)
);
CREATE INDEX IF NOT EXISTS idx_bom_items_bom ON bom_items(bom_id);

DROP TABLE IF EXISTS bom_headers;

COMMIT;
