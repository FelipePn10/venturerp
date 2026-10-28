DROP TABLE IF EXISTS recuperacoes_fornecedor;
DROP INDEX IF EXISTS idx_ta_calls_capa;
ALTER TABLE technical_assistance_calls DROP COLUMN IF EXISTS capa_id;
SELECT instalar_conferencia_de_empresa();
