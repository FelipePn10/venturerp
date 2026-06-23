BEGIN;

-- Plano de Corte — fita de borda (moveleiro): quais lados da peça 2D levam fita,
-- o material da fita e (denormalizado) o custo por metro da fita.
ALTER TABLE public.cutting_plan_parts
    ADD COLUMN IF NOT EXISTS edge_top       BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS edge_bottom    BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS edge_left      BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS edge_right     BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS band_item_code BIGINT,
    ADD COLUMN IF NOT EXISTS band_cost_per_m NUMERIC(15,4) NOT NULL DEFAULT 0;

COMMIT;
