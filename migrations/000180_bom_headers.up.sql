BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- BOM header: versioning / status / type for a product's structure (enterprise+).
--
-- Consolidates the thin, parallel `boms`/`bom_items` model into the real BOM
-- (`item_structures` = the lines) + this header (`bom_headers` = version/status/type
-- per item+mask). The old tables are migrated (best-effort) and dropped.
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS bom_headers (
    id         BIGSERIAL PRIMARY KEY,
    item_code  BIGINT NOT NULL,
    mask       VARCHAR(200),
    bom_type   VARCHAR(20) NOT NULL DEFAULT 'MBOM',
    version    INTEGER NOT NULL DEFAULT 1,
    status     VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'APPROVED', 'OBSOLETE')),
    valid_from DATE,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bom_headers_item ON bom_headers(item_code);
CREATE UNIQUE INDEX IF NOT EXISTS idx_bom_headers_unique ON bom_headers(item_code, COALESCE(mask, ''), version);

-- Best-effort migration of the legacy thin header. Legacy rows land as DRAFT
-- (re-approve as needed); mask lookup is skipped (→ generic). Kept as a simple
-- INSERT…SELECT so the sqlc schema parser accepts it.
INSERT INTO bom_headers (item_code, bom_type, version, status, valid_from, created_by)
SELECT product_id, bom_type, version, 'DRAFT', valid_from, gen_random_uuid()
FROM boms;

DROP TABLE IF EXISTS bom_items;
DROP TABLE IF EXISTS boms;

COMMIT;
