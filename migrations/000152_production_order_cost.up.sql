BEGIN;

-- Apuração de custo real da Ordem de Produção (OF).
-- Material real é valorizado pelo custo médio dos consumos; a conversão
-- (mão-de-obra + overhead aplicado) vem das horas apontadas × custo/hora do
-- centro de trabalho. As variâncias são (real − padrão) por componente.
CREATE TABLE IF NOT EXISTS public.production_order_costs (
    id                  BIGSERIAL PRIMARY KEY,
    production_order_id BIGINT NOT NULL UNIQUE REFERENCES public.production_orders(id),
    produced_qty        NUMERIC(15,4) NOT NULL DEFAULT 0,

    -- custo real apurado (total da OF)
    material_cost_real  NUMERIC(20,6) NOT NULL DEFAULT 0,
    labor_cost_real     NUMERIC(20,6) NOT NULL DEFAULT 0,
    overhead_cost_real  NUMERIC(20,6) NOT NULL DEFAULT 0,
    total_cost_real     NUMERIC(20,6) NOT NULL DEFAULT 0,
    unit_cost_real      NUMERIC(20,6) NOT NULL DEFAULT 0,

    -- custo padrão snapshot (custo unitário padrão × quantidade produzida)
    material_cost_std   NUMERIC(20,6) NOT NULL DEFAULT 0,
    labor_cost_std      NUMERIC(20,6) NOT NULL DEFAULT 0,
    overhead_cost_std   NUMERIC(20,6) NOT NULL DEFAULT 0,
    total_cost_std      NUMERIC(20,6) NOT NULL DEFAULT 0,

    -- variâncias (real − padrão); positivo = gastou mais que o padrão
    material_variance   NUMERIC(20,6) NOT NULL DEFAULT 0,
    labor_variance      NUMERIC(20,6) NOT NULL DEFAULT 0,
    overhead_variance   NUMERIC(20,6) NOT NULL DEFAULT 0,
    total_variance      NUMERIC(20,6) NOT NULL DEFAULT 0,

    currency            VARCHAR(3) NOT NULL DEFAULT 'BRL',
    settled_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    settled_by          UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_poc_order ON public.production_order_costs(production_order_id);

COMMIT;
