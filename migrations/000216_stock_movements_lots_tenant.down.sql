DROP INDEX IF EXISTS uq_stock_lot_balances_tenant;
DROP INDEX IF EXISTS idx_stock_movements_tenant_reference;

ALTER TABLE stock_lot_balances
    ADD CONSTRAINT stock_lot_balances_item_code_mask_warehouse_id_lot_key
    UNIQUE (item_code, mask, warehouse_id, lot);

ALTER TABLE stock_lot_balances DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE stock_movements DROP COLUMN IF EXISTS enterprise_id;
