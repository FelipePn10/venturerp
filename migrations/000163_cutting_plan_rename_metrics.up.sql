BEGIN;

-- Plano de Corte — limpeza de nomenclatura: as métricas guardam comprimento (1D)
-- OU área (2D/true-shape); o sufixo "_mm" herdado do 1D era enganoso.
ALTER TABLE public.cutting_plans RENAME COLUMN total_demand_mm TO total_demand;
ALTER TABLE public.cutting_plans RENAME COLUMN total_stock_mm  TO total_stock;

COMMIT;
