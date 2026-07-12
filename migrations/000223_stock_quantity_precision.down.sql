ALTER TABLE stock_reservations ALTER COLUMN quantity TYPE NUMERIC(15,4) USING ROUND(quantity,4)::NUMERIC(15,4);
ALTER TABLE stock_lot_balances ALTER COLUMN quantity TYPE NUMERIC(15,4) USING ROUND(quantity,4)::NUMERIC(15,4);
ALTER TABLE stock_balances ALTER COLUMN safety_stock TYPE NUMERIC(15,4) USING ROUND(safety_stock,4)::NUMERIC(15,4);
ALTER TABLE stock_balances ALTER COLUMN maximum_stock TYPE NUMERIC(15,4) USING ROUND(maximum_stock,4)::NUMERIC(15,4);
ALTER TABLE stock_balances ALTER COLUMN minimum_stock TYPE NUMERIC(15,4) USING ROUND(minimum_stock,4)::NUMERIC(15,4);
ALTER TABLE stock_balances DROP COLUMN available_qty;
ALTER TABLE stock_balances ALTER COLUMN reserved_qty TYPE NUMERIC(15,4) USING ROUND(reserved_qty,4)::NUMERIC(15,4);
ALTER TABLE stock_balances ALTER COLUMN quantity TYPE NUMERIC(15,4) USING ROUND(quantity,4)::NUMERIC(15,4);
ALTER TABLE stock_movements ALTER COLUMN quantity TYPE NUMERIC(15,4) USING ROUND(quantity,4)::NUMERIC(15,4);
ALTER TABLE stock_balances ADD COLUMN available_qty NUMERIC(15,4)
    GENERATED ALWAYS AS (quantity - reserved_qty) STORED;
