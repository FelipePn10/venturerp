DROP INDEX IF EXISTS idx_user_enterprises_authorization;

ALTER TABLE user_enterprises
    DROP CONSTRAINT IF EXISTS user_enterprises_role_check,
    DROP COLUMN IF EXISTS role;
