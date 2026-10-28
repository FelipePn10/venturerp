DROP INDEX IF EXISTS idx_ta_call_items_serie;
ALTER TABLE technical_assistance_call_items
    DROP COLUMN IF EXISTS cobre_deslocamento, DROP COLUMN IF EXISTS cobre_mao_obra, DROP COLUMN IF EXISTS cobre_pecas,
    DROP COLUMN IF EXISTS garantia_inicio, DROP COLUMN IF EXISTS garantia_origem, DROP COLUMN IF EXISTS serie_id;
ALTER TABLE numeros_serie DROP COLUMN IF EXISTS garantia_estendida_motivo, DROP COLUMN IF EXISTS garantia_estendida_ate;
ALTER TABLE item_pos_venda
    DROP COLUMN IF EXISTS cobre_deslocamento, DROP COLUMN IF EXISTS cobre_mao_obra, DROP COLUMN IF EXISTS cobre_pecas,
    DROP COLUMN IF EXISTS garantia_inicio, DROP COLUMN IF EXISTS garantia_meses;
SELECT instalar_conferencia_de_empresa();
