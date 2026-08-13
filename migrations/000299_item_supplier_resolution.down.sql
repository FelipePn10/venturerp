BEGIN;
ALTER TABLE fiscal_entry_items DROP CONSTRAINT IF EXISTS chk_fiscal_item_resolution_strategy;
ALTER TABLE fiscal_entry_items DROP COLUMN IF EXISTS resolved_at, DROP COLUMN IF EXISTS resolution_strategy, DROP COLUMN IF EXISTS supplier_item_identifier, DROP COLUMN IF EXISTS item_supplier_id;
DROP INDEX IF EXISTS uq_item_supplier_external_code_active;
ALTER TABLE item_preferred_suppliers DROP COLUMN IF EXISTS valid_from;
COMMIT;
