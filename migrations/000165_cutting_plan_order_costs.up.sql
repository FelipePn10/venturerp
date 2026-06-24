BEGIN;

-- Plano de Corte — rateio do custo da baixa por ordem (OP) quando um plano agrega
-- várias ordens. Preenchido ao firmar, proporcional à demanda de cada ordem.
CREATE TABLE IF NOT EXISTS public.cutting_plan_order_costs (
    id             BIGSERIAL PRIMARY KEY,
    plan_id        BIGINT NOT NULL REFERENCES public.cutting_plans(id) ON DELETE CASCADE,
    order_ref      VARCHAR(40) NOT NULL,           -- ex.: OP-1234 / PLAN-88
    demand_measure NUMERIC(18,4) NOT NULL DEFAULT 0, -- comprimento (1D) ou área (2D)
    allocated_cost NUMERIC(15,4) NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cutting_plan_order_costs_plan ON public.cutting_plan_order_costs(plan_id);

COMMIT;
