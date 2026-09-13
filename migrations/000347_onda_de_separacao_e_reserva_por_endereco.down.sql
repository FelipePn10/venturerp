DROP INDEX IF EXISTS idx_picking_waves_st;
DROP INDEX IF EXISTS idx_wave_lines_onda;
DROP TABLE IF EXISTS stock_picking_wave_lines;
DROP TABLE IF EXISTS stock_picking_waves;
DROP INDEX IF EXISTS idx_stock_reservations_endereco;
ALTER TABLE stock_lot_balances DROP COLUMN IF EXISTS reserved_qty;
ALTER TABLE stock_reservations DROP COLUMN IF EXISTS lot, DROP COLUMN IF EXISTS address;
