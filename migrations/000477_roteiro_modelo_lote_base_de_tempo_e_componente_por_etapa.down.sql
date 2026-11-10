DROP FUNCTION IF EXISTS erp_roteiro_vigente(BIGINT, BIGINT, TEXT, DATE, NUMERIC);
CREATE OR REPLACE FUNCTION erp_roteiro_vigente(
    p_enterprise_id BIGINT,
    p_item_code     BIGINT,
    p_mask          TEXT,
    p_data          DATE
) RETURNS BIGINT
LANGUAGE sql STABLE AS $$
    SELECT mr.id
    FROM manufacturing_routes mr
    WHERE mr.enterprise_id = p_enterprise_id
      AND mr.item_code = p_item_code
      AND mr.is_active
      AND mr.situation = 'APROVADA'
      AND (COALESCE(mr.mask, '') = COALESCE(p_mask, '') OR COALESCE(mr.mask, '') = '')
      AND (mr.valid_from IS NULL OR mr.valid_from <= COALESCE(p_data, CURRENT_DATE))
      AND (mr.valid_to   IS NULL OR mr.valid_to   >= COALESCE(p_data, CURRENT_DATE))
    ORDER BY (COALESCE(mr.mask, '') = COALESCE(p_mask, '')) DESC,
             mr.is_standard DESC, mr.valid_from DESC NULLS LAST, mr.alternative
    LIMIT 1
$$;

ALTER TABLE production_order_materials DROP COLUMN IF EXISTS route_operation_sequence;
ALTER TABLE item_structures DROP CONSTRAINT IF EXISTS chk_item_structures_etapa;
ALTER TABLE item_structures DROP COLUMN IF EXISTS route_operation_sequence;

ALTER TABLE route_operations DROP CONSTRAINT IF EXISTS chk_route_ops_time_basis;
ALTER TABLE route_operations DROP COLUMN IF EXISTS time_basis_attribute;
ALTER TABLE route_operations DROP COLUMN IF EXISTS time_basis;
ALTER TABLE operations DROP CONSTRAINT IF EXISTS chk_operations_time_basis;
ALTER TABLE operations DROP COLUMN IF EXISTS time_basis_attribute;
ALTER TABLE operations DROP COLUMN IF EXISTS time_basis;

ALTER TABLE manufacturing_routes DROP CONSTRAINT IF EXISTS chk_mfg_routes_faixa_de_lote;
ALTER TABLE manufacturing_routes DROP COLUMN IF EXISTS lot_max;
ALTER TABLE manufacturing_routes DROP COLUMN IF EXISTS lot_min;

-- Os modelos (sem item) não cabem no esquema antigo: saem com suas etapas.
DELETE FROM manufacturing_routes WHERE is_template;
DROP INDEX IF EXISTS idx_mfg_routes_modelos;
ALTER TABLE manufacturing_routes DROP CONSTRAINT IF EXISTS chk_mfg_routes_modelo_descrito;
ALTER TABLE manufacturing_routes DROP CONSTRAINT IF EXISTS chk_mfg_routes_modelo_sem_item;
ALTER TABLE manufacturing_routes DROP COLUMN IF EXISTS is_template;
ALTER TABLE manufacturing_routes ALTER COLUMN item_code SET NOT NULL;
