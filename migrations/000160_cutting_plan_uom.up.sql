BEGIN;

-- Plano de Corte — unidade de estoque do material e fator de conversão.
-- O corte trabalha em mm, mas o material pode ser estocado em metro, m², m³, peça,
-- kg etc. Estes campos guardam a UoM de estoque (snapshot, copiado do item) e um
-- fator "quantidade de estoque por metro linear" usado para massa/área/volume
-- (densidade linear kg/m, largura m²/m, seção m³/m). Para UoM de comprimento e
-- peça a conversão é geométrica e o fator é ignorado (0).
ALTER TABLE public.cutting_plans
    ADD COLUMN IF NOT EXISTS stock_uom   VARCHAR(12) NOT NULL DEFAULT 'UN',
    ADD COLUMN IF NOT EXISTS uom_factor  NUMERIC(15,6) NOT NULL DEFAULT 0;

COMMIT;
