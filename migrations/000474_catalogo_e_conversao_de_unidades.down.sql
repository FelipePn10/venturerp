BEGIN;

ALTER TABLE fiscal_entry_items
    DROP COLUMN IF EXISTS caminho_conversao,
    DROP COLUMN IF EXISTS origem_conversao,
    DROP COLUMN IF EXISTS unidade_normalizada,
    DROP COLUMN IF EXISTS quantidade_tributavel,
    DROP COLUMN IF EXISTS unidade_tributavel;

DROP TABLE IF EXISTS item_unit_conversion_events;
DROP TABLE IF EXISTS item_lot_unit_conversions;

DROP TRIGGER IF EXISTS trg_aa_item_conversion_fracao ON item_unit_conversions;
DROP TRIGGER IF EXISTS trg_aa_global_conversion_fracao ON global_unit_conversions;
DROP FUNCTION IF EXISTS erp_conversao_fracao_e_fator();
DROP TRIGGER IF EXISTS trg_item_conversion_sem_sobreposicao ON item_unit_conversions;
DROP FUNCTION IF EXISTS erp_item_conversion_sem_sobreposicao();
DROP INDEX IF EXISTS ix_item_unit_conversions_lookup;
DROP INDEX IF EXISTS uq_item_unit_conversions_version;
-- Versões encerradas não cabem no índice antigo (um fator por par).
DELETE FROM item_unit_conversions WHERE NOT is_active OR valid_to IS NOT NULL;
DELETE FROM item_unit_conversions c USING item_unit_conversions n
 WHERE c.item_code = n.item_code AND c.mask = n.mask AND c.from_uom = n.from_uom AND c.to_uom = n.to_uom
   AND c.valid_from < n.valid_from;
CREATE UNIQUE INDEX uq_item_unit_conversions_config ON item_unit_conversions(item_code, mask, from_uom, to_uom);
ALTER TABLE item_unit_conversions
    DROP CONSTRAINT IF EXISTS chk_item_conv_validity,
    DROP CONSTRAINT IF EXISTS chk_item_conv_quantities,
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS notes,
    DROP COLUMN IF EXISTS origin,
    DROP COLUMN IF EXISTS valid_to,
    DROP COLUMN IF EXISTS valid_from,
    DROP COLUMN IF EXISTS to_quantity,
    DROP COLUMN IF EXISTS from_quantity;

-- Volta ao enum: unidade fora dos 15 valores originais cai em UN.
CREATE TYPE unit_of_measurement_enum AS ENUM ('MM','CM','M','IN','KG','M2','M3','UN','MICROMETRO','TONELADA','L','CX','PC','GL','PAR');
ALTER TABLE items DROP CONSTRAINT IF EXISTS fk_items_warehouse_uom;
ALTER TABLE items ALTER COLUMN warehouse_unit_of_measurement DROP DEFAULT;
ALTER TABLE items ALTER COLUMN warehouse_unit_of_measurement TYPE unit_of_measurement_enum
    USING (CASE WHEN warehouse_unit_of_measurement IN ('MM','CM','M','IN','KG','M2','M3','UN','MICROMETRO','TONELADA','L','CX','PC','GL','PAR')
                THEN warehouse_unit_of_measurement ELSE 'UN' END)::unit_of_measurement_enum;
ALTER TABLE items ALTER COLUMN warehouse_unit_of_measurement SET DEFAULT 'UN';
ALTER TABLE item_structures ALTER COLUMN unit_of_measurement DROP DEFAULT;
ALTER TABLE item_structures ALTER COLUMN unit_of_measurement TYPE unit_of_measurement_enum
    USING (CASE WHEN unit_of_measurement IN ('MM','CM','M','IN','KG','M2','M3','UN','MICROMETRO','TONELADA','L','CX','PC','GL','PAR')
                THEN unit_of_measurement ELSE 'UN' END)::unit_of_measurement_enum;
ALTER TABLE item_structures ALTER COLUMN unit_of_measurement SET DEFAULT 'UN';

DROP TRIGGER IF EXISTS trg_enterprise_units_of_measure ON enterprise;
DROP FUNCTION IF EXISTS erp_semear_unidades_nova_empresa();
DROP FUNCTION IF EXISTS erp_semear_unidades_de_medida(BIGINT);

DELETE FROM global_unit_conversions WHERE is_system;
ALTER TABLE global_unit_conversions
    DROP CONSTRAINT IF EXISTS chk_global_conv_quantities,
    DROP COLUMN IF EXISTS is_system,
    DROP COLUMN IF EXISTS to_quantity,
    DROP COLUMN IF EXISTS from_quantity,
    ALTER COLUMN factor TYPE NUMERIC(18,8);

DROP TABLE IF EXISTS unit_of_measure_aliases;
DROP TABLE IF EXISTS units_of_measure;

COMMIT;
