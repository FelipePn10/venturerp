ALTER TABLE items
    DROP COLUMN IF EXISTS abc_calculated_at,
    DROP COLUMN IF EXISTS abc_share_pct,
    DROP COLUMN IF EXISTS abc_consumption_value;
DROP TABLE IF EXISTS stock_abc_count_policy;
DROP INDEX IF EXISTS idx_wh_addresses_item_fixo;
DROP INDEX IF EXISTS idx_wh_addresses_zona;
ALTER TABLE manufacturing_warehouse_addresses
    DROP COLUMN IF EXISTS pick_sequence,
    DROP COLUMN IF EXISTS fixed_item_code,
    DROP COLUMN IF EXISTS block_reason,
    DROP COLUMN IF EXISTS is_blocked,
    DROP COLUMN IF EXISTS capacity,
    DROP COLUMN IF EXISTS zone;
