BEGIN;

-- Plano de Corte — Fase 4: corte true-shape (irregular, laser/plasma).
-- A peça pode ter um contorno (polígono) armazenado como JSON; a caixa
-- envolvente fica em width_mm/height_mm (reaproveitadas da fase 3). A posição
-- ganha um ângulo de rotação livre (graus) para o nesting irregular.

ALTER TABLE public.cutting_plan_parts
    ADD COLUMN IF NOT EXISTS geometry TEXT;

ALTER TABLE public.cutting_pattern_placements
    ADD COLUMN IF NOT EXISTS rotation_deg NUMERIC(9,4) NOT NULL DEFAULT 0;

COMMIT;
