BEGIN;
ALTER TABLE fiscal_entry_items DROP CONSTRAINT IF EXISTS chk_fiscal_item_resolution_strategy;
ALTER TABLE fiscal_entry_items ADD CONSTRAINT chk_fiscal_item_resolution_strategy
 CHECK(resolution_strategy IS NULL OR resolution_strategy IN ('CODIGO_EXATO','DESCRICAO','MANUAL','NAO_RESOLVIDO'));
CREATE INDEX idx_fiscal_entry_items_supplier_link ON fiscal_entry_items(item_supplier_id) WHERE item_supplier_id IS NOT NULL;
COMMIT;
