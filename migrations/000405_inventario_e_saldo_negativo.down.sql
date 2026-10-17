ALTER TABLE warehouse DROP COLUMN IF EXISTS allow_negative_stock;
ALTER TABLE stock_cycle_counts DROP COLUMN IF EXISTS adjustment_movement_id;
ALTER TABLE physical_inventories DROP COLUMN IF EXISTS adjustment_value, DROP COLUMN IF EXISTS closed_by;
DROP INDEX IF EXISTS uq_physical_inventory_items_item;
ALTER TABLE physical_inventory_items
    DROP COLUMN IF EXISTS adjustment_movement_id,
    DROP COLUMN IF EXISTS previous_counted_qty,
    DROP COLUMN IF EXISTS count_number;
