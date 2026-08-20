BEGIN;
ALTER TABLE notification_event_catalog DROP CONSTRAINT IF EXISTS notification_event_catalog_producer_status_check;
ALTER TABLE notification_event_catalog DROP COLUMN IF EXISTS producer_description;
ALTER TABLE notification_event_catalog DROP COLUMN IF EXISTS producer_status;
COMMIT;
