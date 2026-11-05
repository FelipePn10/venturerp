DROP TABLE IF EXISTS stock_cost_recalc_log;
DROP TABLE IF EXISTS stock_cost_variances;
DROP TABLE IF EXISTS stock_closing_balances;
DROP TABLE IF EXISTS stock_closings;
DROP INDEX IF EXISTS idx_stock_movements_retroativos;
DROP INDEX IF EXISTS idx_stock_movements_data;
ALTER TABLE stock_movements DROP COLUMN backdated, DROP COLUMN movement_date;
