BEGIN;

ALTER TABLE public.cutting_plans RENAME COLUMN total_demand TO total_demand_mm;
ALTER TABLE public.cutting_plans RENAME COLUMN total_stock  TO total_stock_mm;

COMMIT;
