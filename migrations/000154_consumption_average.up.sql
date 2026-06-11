BEGIN;

-- Consumo médio mensal por item, calculado a partir das saídas de estoque numa
-- janela móvel. Alimenta o ponto de reposição (ROP) do MRP sem digitação manual.
CREATE TABLE IF NOT EXISTS public.item_consumption_averages (
    id                      BIGSERIAL PRIMARY KEY,
    item_code               BIGINT NOT NULL UNIQUE,
    avg_monthly_consumption NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_consumed          NUMERIC(15,4) NOT NULL DEFAULT 0,
    window_months           INT NOT NULL DEFAULT 6,
    calculated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_item_consumption_averages_item
    ON public.item_consumption_averages(item_code);

COMMIT;
