ALTER TABLE items
    ALTER COLUMN warehouse_avg_monthly_consumption_manual TYPE INTEGER USING ROUND(warehouse_avg_monthly_consumption_manual)::INTEGER;
