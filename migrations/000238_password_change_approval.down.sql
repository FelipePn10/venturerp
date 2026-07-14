DROP TABLE IF EXISTS password_change_requests;
ALTER TABLE users DROP COLUMN IF EXISTS auth_version;
