ALTER TABLE item_consumption_averages
    ALTER COLUMN avg_monthly_consumption TYPE NUMERIC(15,4),
    ALTER COLUMN total_consumed          TYPE NUMERIC(15,4);
ALTER TABLE physical_inventory_items ALTER COLUMN unit_cost TYPE NUMERIC(20,6);
ALTER TABLE stock_lot_balances ALTER COLUMN last_cost TYPE NUMERIC(15,4);
ALTER TABLE stock_balances
    ALTER COLUMN avg_cost  TYPE NUMERIC(15,4),
    ALTER COLUMN last_cost TYPE NUMERIC(15,4);
ALTER TABLE stock_movements ALTER COLUMN unit_price TYPE NUMERIC(15,4);
