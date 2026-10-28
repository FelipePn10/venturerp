ALTER TABLE quality_lot_results DROP COLUMN IF EXISTS instrument_id;
DROP TRIGGER IF EXISTS trg_quality_calibrations_imutaveis ON quality_calibrations;
DROP FUNCTION IF EXISTS quality_calibrations_imutaveis();
DROP TABLE IF EXISTS quality_calibrations;
DROP TABLE IF EXISTS quality_instruments;
SELECT instalar_conferencia_de_empresa();
