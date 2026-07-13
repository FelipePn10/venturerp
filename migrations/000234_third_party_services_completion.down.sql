BEGIN;
DROP TABLE IF EXISTS third_party_service_order_history;
DROP INDEX IF EXISTS uq_item_unit_conversions_config;
DELETE FROM item_unit_conversions WHERE mask<>'';
ALTER TABLE item_unit_conversions
    DROP COLUMN IF EXISTS tolerance_type,
    DROP COLUMN IF EXISTS tolerance_value,
    DROP COLUMN IF EXISTS rounding_percent,
    DROP COLUMN IF EXISTS mask;
ALTER TABLE item_unit_conversions ADD CONSTRAINT item_unit_conversions_item_code_from_uom_to_uom_key UNIQUE(item_code,from_uom,to_uom);
DROP INDEX IF EXISTS uq_third_party_service_movement_idempotency;
ALTER TABLE third_party_service_movements
    DROP COLUMN IF EXISTS lot,
    DROP COLUMN IF EXISTS warehouse_id,
    DROP COLUMN IF EXISTS idempotency_key;
DROP TABLE IF EXISTS global_unit_conversions;
COMMIT;
