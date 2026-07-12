DROP INDEX IF EXISTS uq_drawings_tenant_code;
DROP INDEX IF EXISTS idx_drawings_tenant_item;
ALTER TABLE drawings ADD CONSTRAINT drawings_code_digit_format_key UNIQUE (code, digit, format);
ALTER TABLE drawings DROP COLUMN IF EXISTS enterprise_id;
