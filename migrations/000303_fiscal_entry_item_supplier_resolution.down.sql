BEGIN;
DROP INDEX IF EXISTS idx_fiscal_entry_items_supplier_link;
ALTER TABLE fiscal_entry_items DROP CONSTRAINT IF EXISTS chk_fiscal_item_resolution_strategy;
DO $$ BEGIN IF EXISTS(SELECT 1 FROM fiscal_entry_items WHERE resolution_strategy IN ('CODIGO_EXATO','DESCRICAO','NAO_RESOLVIDO')) THEN RAISE EXCEPTION 'Rollback inseguro: existem resolucoes no formato novo'; END IF; END $$;
ALTER TABLE fiscal_entry_items ADD CONSTRAINT chk_fiscal_item_resolution_strategy CHECK (resolution_strategy IS NULL OR resolution_strategy IN ('CODIGO_FORNECEDOR','DESCRICAO_FORNECEDOR','MANUAL'));
COMMIT;
