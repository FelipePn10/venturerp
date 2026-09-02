BEGIN;
DROP INDEX IF EXISTS idx_bom_headers_tenant_item;
DROP INDEX IF EXISTS idx_bom_headers_unique;
ALTER TABLE bom_headers DROP COLUMN IF EXISTS enterprise_id;
CREATE UNIQUE INDEX IF NOT EXISTS idx_bom_headers_unique
    ON bom_headers(item_code, COALESCE(mask, ''), version);
COMMIT;
