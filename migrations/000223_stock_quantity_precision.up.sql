ALTER TABLE stock_movements ALTER COLUMN quantity TYPE NUMERIC(18,6) USING quantity::NUMERIC(18,6);
ALTER TABLE stock_balances DROP COLUMN available_qty;
ALTER TABLE stock_balances ALTER COLUMN quantity TYPE NUMERIC(18,6) USING quantity::NUMERIC(18,6);
ALTER TABLE stock_balances ALTER COLUMN reserved_qty TYPE NUMERIC(18,6) USING reserved_qty::NUMERIC(18,6);
ALTER TABLE stock_balances ALTER COLUMN minimum_stock TYPE NUMERIC(18,6) USING minimum_stock::NUMERIC(18,6);
ALTER TABLE stock_balances ALTER COLUMN maximum_stock TYPE NUMERIC(18,6) USING maximum_stock::NUMERIC(18,6);
ALTER TABLE stock_balances ALTER COLUMN safety_stock TYPE NUMERIC(18,6) USING safety_stock::NUMERIC(18,6);
ALTER TABLE stock_lot_balances ALTER COLUMN quantity TYPE NUMERIC(18,6) USING quantity::NUMERIC(18,6);
ALTER TABLE stock_reservations ALTER COLUMN quantity TYPE NUMERIC(18,6) USING quantity::NUMERIC(18,6);
ALTER TABLE stock_balances ADD COLUMN available_qty NUMERIC(18,6)
    GENERATED ALWAYS AS (quantity - reserved_qty) STORED;
