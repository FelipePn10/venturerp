DROP INDEX IF EXISTS idx_item_structures_customer_supplied;
ALTER TABLE item_structures DROP COLUMN IF EXISTS customer_supplied;
