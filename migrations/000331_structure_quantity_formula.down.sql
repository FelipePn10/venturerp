BEGIN;
DROP INDEX IF EXISTS idx_item_structures_quantity_formula;
ALTER TABLE item_structures DROP COLUMN IF EXISTS quantity_scale;
ALTER TABLE item_structures DROP COLUMN IF EXISTS quantity_rounding;
ALTER TABLE item_structures DROP COLUMN IF EXISTS quantity_formula;
COMMIT;
