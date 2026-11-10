-- Roteiro de fabricação: o que faltava em relação a SAP/Oracle/Senior e cabe
-- em metalúrgica e moveleira.

-- ── 1. Roteiro-modelo ───────────────────────────────────────────────────────
-- Um roteiro sem item: a sequência típica de uma família de peças ("chapa
-- cortada, dobrada e pintada"), usada para criar o roteiro de um item novo por
-- cópia, no modo Modelos da VENT0202. O modelo nunca entra em OP, MRP, CRP nem custo: todas
-- as escolhas de roteiro filtram por item, e item nulo não casa com nada.
ALTER TABLE manufacturing_routes ALTER COLUMN item_code DROP NOT NULL;
ALTER TABLE manufacturing_routes ADD COLUMN IF NOT EXISTS is_template BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE manufacturing_routes ADD CONSTRAINT chk_mfg_routes_modelo_sem_item
    CHECK ((is_template AND item_code IS NULL AND mask IS NULL AND NOT is_standard)
        OR (NOT is_template AND item_code IS NOT NULL));
ALTER TABLE manufacturing_routes ADD CONSTRAINT chk_mfg_routes_modelo_descrito
    CHECK (NOT is_template OR length(btrim(COALESCE(description, ''))) > 0);
CREATE INDEX IF NOT EXISTS idx_mfg_routes_modelos
    ON manufacturing_routes (enterprise_id, is_active) WHERE is_template;

-- ── 2. Faixa de lote por roteiro ────────────────────────────────────────────
-- Roteiro alternativo escolhido pela quantidade (SAP: "lot size from/to"):
-- abaixo de 50 peças a guilhotina, a partir de 50 o laser. Sem faixa, o
-- roteiro vale para qualquer quantidade.
ALTER TABLE manufacturing_routes ADD COLUMN IF NOT EXISTS lot_min NUMERIC(15,4);
ALTER TABLE manufacturing_routes ADD COLUMN IF NOT EXISTS lot_max NUMERIC(15,4);
ALTER TABLE manufacturing_routes ADD CONSTRAINT chk_mfg_routes_faixa_de_lote
    CHECK ((lot_min IS NULL OR lot_min > 0)
       AND (lot_max IS NULL OR lot_max > 0)
       AND (lot_min IS NULL OR lot_max IS NULL OR lot_max >= lot_min));

-- ── 3. Base do tempo ────────────────────────────────────────────────────────
-- O tempo da operação pode ser POR PEÇA (o de sempre) ou por uma medida da
-- peça, lida das dimensões de engenharia do item:
--   M2            comprimento × largura          (pintura, lixamento, laminação)
--   METRO_LINEAR  comprimento                    (corte de tubo, solda contínua)
--   PERIMETRO     2 × (comprimento + largura)    (fita de borda, rebarbação)
--   ATRIBUTO      um atributo numérico do PDM    (furos, metros de fita, dobras)
-- O tempo de máquina passa a ser "por N unidades da base" (N = peças por ciclo).
ALTER TABLE operations ADD COLUMN IF NOT EXISTS time_basis VARCHAR(20) NOT NULL DEFAULT 'PECA';
ALTER TABLE operations ADD COLUMN IF NOT EXISTS time_basis_attribute VARCHAR(60);
ALTER TABLE operations ADD CONSTRAINT chk_operations_time_basis
    CHECK (time_basis IN ('PECA','M2','METRO_LINEAR','PERIMETRO','ATRIBUTO')
       AND (time_basis <> 'ATRIBUTO' OR length(btrim(COALESCE(time_basis_attribute, ''))) > 0));
ALTER TABLE route_operations ADD COLUMN IF NOT EXISTS time_basis VARCHAR(20);
ALTER TABLE route_operations ADD COLUMN IF NOT EXISTS time_basis_attribute VARCHAR(60);
ALTER TABLE route_operations ADD CONSTRAINT chk_route_ops_time_basis
    CHECK (time_basis IS NULL
        OR (time_basis IN ('PECA','M2','METRO_LINEAR','PERIMETRO','ATRIBUTO')
            AND (time_basis <> 'ATRIBUTO' OR length(btrim(COALESCE(time_basis_attribute, ''))) > 0)));

-- ── 4. Componente consumido numa etapa ──────────────────────────────────────
-- A linha da estrutura diz em que etapa do roteiro do PAI o componente entra
-- (SAP/Oracle: alocação de componente à operação). A ferragem só é requisitada
-- quando a montagem começa, não no corte. Guardamos a SEQUÊNCIA, não o id da
-- etapa: ela sobrevive a roteiro alternativo e a cópia. Nula = entra no início.
ALTER TABLE item_structures ADD COLUMN IF NOT EXISTS route_operation_sequence SMALLINT;
ALTER TABLE item_structures ADD CONSTRAINT chk_item_structures_etapa
    CHECK (route_operation_sequence IS NULL OR route_operation_sequence > 0);
ALTER TABLE production_order_materials ADD COLUMN IF NOT EXISTS route_operation_sequence SMALLINT;

-- ── 5. Regra única de escolha, agora com a quantidade ───────────────────────
DROP FUNCTION IF EXISTS erp_roteiro_vigente(BIGINT, BIGINT, TEXT, DATE);
CREATE OR REPLACE FUNCTION erp_roteiro_vigente(
    p_enterprise_id BIGINT,
    p_item_code     BIGINT,
    p_mask          TEXT,
    p_data          DATE,
    p_quantidade    NUMERIC DEFAULT NULL
) RETURNS BIGINT
LANGUAGE sql STABLE AS $$
    SELECT mr.id
    FROM manufacturing_routes mr
    WHERE mr.enterprise_id = p_enterprise_id
      AND mr.item_code = p_item_code
      AND NOT mr.is_template
      AND mr.is_active
      AND mr.situation = 'APROVADA'
      AND (COALESCE(mr.mask, '') = COALESCE(p_mask, '') OR COALESCE(mr.mask, '') = '')
      AND (mr.valid_from IS NULL OR mr.valid_from <= COALESCE(p_data, CURRENT_DATE))
      AND (mr.valid_to   IS NULL OR mr.valid_to   >= COALESCE(p_data, CURRENT_DATE))
      AND (p_quantidade IS NULL OR mr.lot_min IS NULL OR p_quantidade >= mr.lot_min)
      AND (p_quantidade IS NULL OR mr.lot_max IS NULL OR p_quantidade <= mr.lot_max)
    ORDER BY (COALESCE(mr.mask, '') = COALESCE(p_mask, '')) DESC,
             (p_quantidade IS NOT NULL AND (mr.lot_min IS NOT NULL OR mr.lot_max IS NOT NULL)) DESC,
             mr.is_standard DESC, mr.valid_from DESC NULLS LAST, mr.alternative
    LIMIT 1
$$;

COMMENT ON FUNCTION erp_roteiro_vigente(BIGINT, BIGINT, TEXT, DATE, NUMERIC) IS
    'Roteiro que vale para item/máscara/quantidade na data: máscara exata vence o genérico; faixa de lote que casa vence o roteiro sem faixa; depois padrão, vigência mais recente, menor alternativa.';
