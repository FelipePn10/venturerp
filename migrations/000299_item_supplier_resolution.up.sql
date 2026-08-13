BEGIN;
ALTER TABLE item_preferred_suppliers ADD COLUMN IF NOT EXISTS valid_from DATE;
CREATE UNIQUE INDEX IF NOT EXISTS uq_item_supplier_external_code_active
ON item_preferred_suppliers(enterprise_id,supplier_code,upper(btrim(supplier_item_code)))
WHERE is_active AND supplier_item_code IS NOT NULL AND btrim(supplier_item_code)<>'';
ALTER TABLE fiscal_entry_items
    ADD COLUMN IF NOT EXISTS item_supplier_id BIGINT REFERENCES item_preferred_suppliers(id),
    ADD COLUMN IF NOT EXISTS supplier_item_identifier VARCHAR(255),
    ADD COLUMN IF NOT EXISTS resolution_strategy VARCHAR(30),
    ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ;
ALTER TABLE fiscal_entry_items ADD CONSTRAINT chk_fiscal_item_resolution_strategy
CHECK (resolution_strategy IS NULL OR resolution_strategy IN ('CODIGO_FORNECEDOR','DESCRICAO_FORNECEDOR','MANUAL'));
COMMIT;
