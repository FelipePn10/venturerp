DROP INDEX IF EXISTS idx_stock_lots_fefo;
DROP INDEX IF EXISTS idx_stock_balances_endereco;
DROP INDEX IF EXISTS uq_stock_lot_balances_endereco;
CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_lot_balances_tenant
    ON stock_lot_balances (enterprise_id, item_code, mask, warehouse_id, lot)
 WHERE enterprise_id IS NOT NULL;
DROP INDEX IF EXISTS uq_stock_balances_endereco;
ALTER TABLE stock_lots         DROP COLUMN IF EXISTS expires_at;
ALTER TABLE stock_movements    DROP COLUMN IF EXISTS address_to;
ALTER TABLE stock_movements    DROP COLUMN IF EXISTS address;
ALTER TABLE stock_lot_balances DROP COLUMN IF EXISTS address;
ALTER TABLE stock_balances     DROP COLUMN IF EXISTS address;
