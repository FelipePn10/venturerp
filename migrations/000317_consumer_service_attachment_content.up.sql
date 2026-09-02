ALTER TABLE consumer_service_call_attachments
    ALTER COLUMN file_path DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS file_content BYTEA,
    ADD COLUMN IF NOT EXISTS file_size BIGINT NOT NULL DEFAULT 0;

ALTER TABLE consumer_service_call_attachments
	ADD CONSTRAINT consumer_service_attachment_content_size
	CHECK (
		file_size >= 0
		AND file_size <= 10485760
		AND (file_content IS NULL OR file_size = octet_length(file_content))
	);
