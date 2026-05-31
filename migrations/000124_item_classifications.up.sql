-- ─── Classificação de Itens ───────────────────────────────────────────────────
--
-- Two-level structure:
--   1. Masks define the hierarchical pattern (e.g. "99.999.999.9999")
--   2. Classifications are the actual codes assigned to items following a mask
--      and may have a parent (enabling multi-level hierarchy)

CREATE TABLE item_classification_masks (
    id          BIGSERIAL    PRIMARY KEY,
    code        BIGINT       NOT NULL UNIQUE,
    mask        VARCHAR(50)  NOT NULL UNIQUE,   -- e.g. "99.999.999.9999"
    description VARCHAR(200) NOT NULL,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE item_classifications (
    id          BIGSERIAL    PRIMARY KEY,
    code        VARCHAR(50)  NOT NULL,           -- e.g. "10.100.100"
    mask_id     BIGINT       NOT NULL REFERENCES item_classification_masks(id),
    parent_id   BIGINT       REFERENCES item_classifications(id) ON DELETE RESTRICT,
    level       INT          NOT NULL DEFAULT 1,
    description VARCHAR(200) NOT NULL,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (mask_id, code)
);

CREATE INDEX idx_item_classifications_mask   ON item_classifications (mask_id);
CREATE INDEX idx_item_classifications_parent ON item_classifications (parent_id) WHERE parent_id IS NOT NULL;
