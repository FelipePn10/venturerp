BEGIN;

-- Fórmula de quantidade por componente (FENG0210 do Focco e equivalentes SAP/TOTVS).
-- Quando preenchida, a quantidade do componente é calculada a partir das
-- variáveis do configurador do pai — por exemplo
-- "2*(COMPRIMENTO/1000)+2*(PROFUNDIDADE/1000)". A coluna quantity permanece
-- como valor nominal/fallback quando a fórmula não puder ser avaliada.
ALTER TABLE item_structures ADD COLUMN IF NOT EXISTS quantity_formula VARCHAR(500);

-- Arredondamento do resultado da fórmula: NONE (sem), UP (para cima),
-- DOWN (para baixo) ou NEAREST (mais próximo), com o número de casas decimais.
ALTER TABLE item_structures ADD COLUMN IF NOT EXISTS quantity_rounding VARCHAR(10) NOT NULL DEFAULT 'NONE'
    CONSTRAINT chk_structure_qty_rounding CHECK (quantity_rounding IN ('NONE','UP','DOWN','NEAREST'));
ALTER TABLE item_structures ADD COLUMN IF NOT EXISTS quantity_scale SMALLINT NOT NULL DEFAULT 4
    CONSTRAINT chk_structure_qty_scale CHECK (quantity_scale BETWEEN 0 AND 6);

CREATE INDEX IF NOT EXISTS idx_item_structures_quantity_formula
    ON item_structures(parent_code) WHERE quantity_formula IS NOT NULL;

COMMIT;
