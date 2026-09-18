ALTER TABLE item_structures DROP CONSTRAINT IF EXISTS chk_item_structures_conversion;
ALTER TABLE item_structures
    DROP COLUMN IF EXISTS quantity_stock_uom,
    DROP COLUMN IF EXISTS conversion_factor;
