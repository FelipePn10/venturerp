DELETE FROM notification_alerts WHERE event_key IN ('COMERCIAL_CONTRATO_VENCENDO', 'COMERCIAL_CONTRATO_SALDO_BAIXO');
DELETE FROM notification_outbox WHERE event_key IN ('COMERCIAL_CONTRATO_VENCENDO', 'COMERCIAL_CONTRATO_SALDO_BAIXO');
DELETE FROM enterprise_notification_subscriptions WHERE event_key IN ('COMERCIAL_CONTRATO_VENCENDO', 'COMERCIAL_CONTRATO_SALDO_BAIXO');
DELETE FROM notification_event_catalog WHERE event_key IN ('COMERCIAL_CONTRATO_VENCENDO', 'COMERCIAL_CONTRATO_SALDO_BAIXO');
DROP TABLE IF EXISTS contratos_venda_chamadas;
DROP TABLE IF EXISTS contratos_venda_programacao;
DROP TABLE IF EXISTS contratos_venda_itens;
DROP TABLE IF EXISTS contratos_venda;
SELECT instalar_conferencia_de_empresa();
