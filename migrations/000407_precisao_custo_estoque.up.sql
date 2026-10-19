-- Precisão do custo unitário do estoque.
--
-- unit_price, avg_cost e last_cost tinham 4 casas. Um item comprado em barra e
-- consumido em milímetro custa frações como R$ 0,0000667 por mm: com 4 casas
-- isso vira 0,0001 (50% a mais) ou zero, e o erro se multiplica por milhares de
-- milímetros em cada ordem. Dez casas cobrem qualquer custo unitário real. Os
-- TOTAIS (total_price, total_cost) são dinheiro e seguem com 4 casas.
--
-- Quantidades já tinham 6 casas; o consumo médio passa de 4 para 6, como elas.
ALTER TABLE stock_movements
    ALTER COLUMN unit_price TYPE NUMERIC(24,10);
ALTER TABLE stock_balances
    ALTER COLUMN avg_cost  TYPE NUMERIC(24,10),
    ALTER COLUMN last_cost TYPE NUMERIC(24,10);
ALTER TABLE stock_lot_balances
    ALTER COLUMN last_cost TYPE NUMERIC(24,10);
ALTER TABLE physical_inventory_items
    ALTER COLUMN unit_cost TYPE NUMERIC(24,10);
ALTER TABLE item_consumption_averages
    ALTER COLUMN avg_monthly_consumption TYPE NUMERIC(20,6),
    ALTER COLUMN total_consumed          TYPE NUMERIC(20,6);
