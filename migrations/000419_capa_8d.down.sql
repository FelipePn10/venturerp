DROP TRIGGER IF EXISTS trg_quality_capa_eventos_imutaveis ON quality_capa_eventos;
DROP FUNCTION IF EXISTS quality_capa_eventos_imutaveis();
DROP TABLE IF EXISTS quality_capa_eventos;
DROP TABLE IF EXISTS quality_capa_acoes;
DROP TABLE IF EXISTS quality_capa_equipe;
DROP TABLE IF EXISTS quality_capa_ncs;
DROP TABLE IF EXISTS quality_capas;
SELECT instalar_conferencia_de_empresa();
