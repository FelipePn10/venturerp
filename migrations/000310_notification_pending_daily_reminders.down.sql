BEGIN;

DROP INDEX IF EXISTS idx_notification_delivery_alert;
ALTER TABLE notification_deliveries DROP COLUMN IF EXISTS alert_id;

COMMIT;
