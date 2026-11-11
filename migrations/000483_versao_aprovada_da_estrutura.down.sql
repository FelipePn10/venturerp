DROP TABLE IF EXISTS bom_header_lines;
DROP INDEX IF EXISTS idx_bom_headers_uma_aprovada;
ALTER TABLE bom_headers
    DROP COLUMN IF EXISTS approved_by,
    DROP COLUMN IF EXISTS approved_at,
    DROP COLUMN IF EXISTS obsoleted_at,
    DROP COLUMN IF EXISTS notes;
