BEGIN;

DROP INDEX IF EXISTS idx_production_order_scrap_scope;
DROP INDEX IF EXISTS idx_manufacturing_warehouse_addresses;
DROP INDEX IF EXISTS idx_manufacturing_stock_closed_periods;

ALTER TABLE production_order_scrap_destinations
    DROP COLUMN IF EXISTS scrap_uom,
    DROP COLUMN IF EXISTS source_uom,
    DROP COLUMN IF EXISTS scrap_quantity,
    DROP COLUMN IF EXISTS return_quantity,
    DROP COLUMN IF EXISTS destination_kind;

DROP TABLE IF EXISTS manufacturing_warehouse_addresses;
DROP TABLE IF EXISTS manufacturing_stock_closed_periods;
DROP TABLE IF EXISTS manufacturing_stock_parameters;
DROP TABLE IF EXISTS manufacturing_stock_item_controls;

ALTER TABLE production_order_materials
    DROP COLUMN IF EXISTS controls_address,
    DROP COLUMN IF EXISTS controls_lot,
    DROP COLUMN IF EXISTS uom;

ALTER TABLE production_orders
    DROP CONSTRAINT IF EXISTS production_order_temp_lot_dates_chk,
    DROP COLUMN IF EXISTS temporary_lot_expires_on,
    DROP COLUMN IF EXISTS temporary_lot_manufactured_on,
    DROP COLUMN IF EXISTS temporary_lot_code;

COMMIT;
