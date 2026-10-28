DROP TABLE IF EXISTS sla_pausas;
ALTER TABLE technical_assistance_calls
    DROP COLUMN IF EXISTS sla_horas_pausadas,
    DROP COLUMN IF EXISTS sla_pausado_desde,
    DROP COLUMN IF EXISTS sla_solucao_risco_em,
    DROP COLUMN IF EXISTS sla_resposta_risco_em,
    DROP COLUMN IF EXISTS sla_horas_uteis;
DROP TABLE IF EXISTS sla_expediente;
ALTER TABLE sla_assistencia DROP COLUMN IF EXISTS horas_uteis;

SELECT instalar_conferencia_de_empresa();
