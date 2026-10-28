ALTER TABLE non_conformances DROP COLUMN IF EXISTS lot_id;
ALTER TABLE inspection_plans DROP COLUMN IF EXISTS sampling_procedure_id;
ALTER TABLE receiving_inspection_routes DROP COLUMN IF EXISTS sampling_procedure_id;
DROP TABLE IF EXISTS quality_lot_supplier_reports;
DROP TRIGGER IF EXISTS trg_quality_lot_decisions_imutaveis ON quality_lot_decisions;
DROP FUNCTION IF EXISTS quality_lot_decisions_imutaveis();
DROP TABLE IF EXISTS quality_lot_decisions;
DROP TABLE IF EXISTS quality_lot_results;
DROP TABLE IF EXISTS quality_lot_characteristics;
DROP TABLE IF EXISTS quality_inspection_lots;
-- Refaz os gatilhos de conferência de empresa sem as colunas removidas.
SELECT instalar_conferencia_de_empresa();
