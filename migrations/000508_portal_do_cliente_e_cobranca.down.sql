DELETE FROM notification_outbox WHERE event_key = 'COMERCIAL_PEDIDO_PORTAL';
DELETE FROM notification_event_catalog WHERE event_key = 'COMERCIAL_PEDIDO_PORTAL';
ALTER TABLE contas_receber DROP COLUMN IF EXISTS cobranca_externa_id;
DROP TABLE IF EXISTS portal_events;
DROP TABLE IF EXISTS portal_links;
SELECT instalar_conferencia_de_empresa();
