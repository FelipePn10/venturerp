DROP INDEX IF EXISTS idx_audit_log_enterprise_occurred;
ALTER TABLE audit_log DROP COLUMN IF EXISTS enterprise_id;
