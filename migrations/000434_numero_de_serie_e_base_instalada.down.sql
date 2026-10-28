DROP TABLE IF EXISTS fiscal_exit_item_series;
DROP TRIGGER IF EXISTS trg_numeros_serie_eventos_imutaveis ON numeros_serie_eventos;
DROP TABLE IF EXISTS numeros_serie_eventos;
DROP FUNCTION IF EXISTS numeros_serie_eventos_imutaveis();
DROP TABLE IF EXISTS numeros_serie;
DROP TABLE IF EXISTS item_pos_venda;
SELECT instalar_conferencia_de_empresa();
