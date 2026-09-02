ALTER TABLE consumer_service_call_attachments
    DROP CONSTRAINT IF EXISTS consumer_service_attachment_content_size,
    DROP COLUMN IF EXISTS file_size,
    DROP COLUMN IF EXISTS file_content;

UPDATE consumer_service_call_attachments SET file_path='' WHERE file_path IS NULL;
ALTER TABLE consumer_service_call_attachments ALTER COLUMN file_path SET NOT NULL;
