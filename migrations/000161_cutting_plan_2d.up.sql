BEGIN;

-- Plano de Corte — Fase 3: corte 2D guilhotinado (chapa / painel MDF).
-- As tabelas 1D ganham as dimensões e atributos 2D; length_mm fica 0 nas linhas 2D.

ALTER TABLE public.cutting_plan_parts
    ADD COLUMN IF NOT EXISTS width_mm       NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS height_mm      NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS grain          VARCHAR(10) NOT NULL DEFAULT 'NONE'
        CHECK (grain IN ('NONE','LENGTH','WIDTH')),
    ADD COLUMN IF NOT EXISTS allow_rotation BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE public.cutting_stock_pieces
    ADD COLUMN IF NOT EXISTS width_mm  NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS height_mm NUMERIC(15,4) NOT NULL DEFAULT 0;

ALTER TABLE public.cutting_patterns
    ADD COLUMN IF NOT EXISTS stock_width_mm    NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS stock_height_mm   NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS used_area_mm2     NUMERIC(18,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS remnant_area_mm2  NUMERIC(18,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS remnant_width_mm  NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS remnant_height_mm NUMERIC(15,4) NOT NULL DEFAULT 0;

ALTER TABLE public.cutting_pattern_placements
    ADD COLUMN IF NOT EXISTS pos_x_mm  NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS pos_y_mm  NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS width_mm  NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS height_mm NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS rotated   BOOLEAN NOT NULL DEFAULT FALSE;

-- Retalhos 2D: além do comprimento (1D), guardam largura/altura.
ALTER TABLE public.stock_remnants
    ADD COLUMN IF NOT EXISTS width_mm  NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS height_mm NUMERIC(15,4) NOT NULL DEFAULT 0;

COMMIT;
