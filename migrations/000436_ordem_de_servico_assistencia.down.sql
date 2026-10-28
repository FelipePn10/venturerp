DROP TRIGGER IF EXISTS trg_os_eventos_imutaveis ON os_eventos;
DROP TABLE IF EXISTS os_eventos;
DROP FUNCTION IF EXISTS os_eventos_imutaveis();
DROP TABLE IF EXISTS os_despesas;
DROP TABLE IF EXISTS os_pecas;
DROP TABLE IF EXISTS os_horas;
DROP TABLE IF EXISTS ordens_servico;
DROP TABLE IF EXISTS os_parametros;
SELECT instalar_conferencia_de_empresa();
