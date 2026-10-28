DELETE FROM notification_alerts WHERE event_key IN ('ASSISTENCIA_SLA_EM_RISCO', 'ASSISTENCIA_SLA_VENCIDO');
DELETE FROM notification_outbox WHERE event_key IN ('ASSISTENCIA_SLA_EM_RISCO', 'ASSISTENCIA_SLA_VENCIDO');
DELETE FROM enterprise_notification_subscriptions WHERE event_key IN ('ASSISTENCIA_SLA_EM_RISCO', 'ASSISTENCIA_SLA_VENCIDO');
DELETE FROM notification_event_catalog WHERE event_key IN ('ASSISTENCIA_SLA_EM_RISCO', 'ASSISTENCIA_SLA_VENCIDO');
DROP INDEX IF EXISTS idx_ta_calls_sla_aberto;
ALTER TABLE technical_assistance_calls
    DROP COLUMN IF EXISTS respondido_em,
    DROP COLUMN IF EXISTS sla_alerta_pct,
    DROP COLUMN IF EXISTS sla_solucao_ate,
    DROP COLUMN IF EXISTS sla_resposta_ate,
    DROP COLUMN IF EXISTS sla_id;
DROP TABLE IF EXISTS sla_assistencia;
SELECT instalar_conferencia_de_empresa();
