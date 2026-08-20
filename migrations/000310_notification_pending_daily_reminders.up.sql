BEGIN;

ALTER TABLE notification_deliveries
    ADD COLUMN alert_id UUID REFERENCES notification_alerts(id) ON DELETE RESTRICT;

CREATE INDEX idx_notification_delivery_alert
    ON notification_deliveries(enterprise_id, alert_id, created_at DESC)
    WHERE alert_id IS NOT NULL;

COMMIT;
