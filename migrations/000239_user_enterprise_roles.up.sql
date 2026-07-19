ALTER TABLE user_enterprises
    ADD COLUMN IF NOT EXISTS role VARCHAR(10);

UPDATE user_enterprises association
SET role = CASE WHEN UPPER(TRIM(COALESCE(app_user.role, ''))) = 'ADMIN' THEN 'ADMIN' ELSE 'USER' END
FROM users app_user
WHERE app_user.id = association.user_id
  AND association.role IS NULL;

ALTER TABLE user_enterprises
    ALTER COLUMN role SET DEFAULT 'USER',
    ALTER COLUMN role SET NOT NULL;

ALTER TABLE user_enterprises
    DROP CONSTRAINT IF EXISTS user_enterprises_role_check;

ALTER TABLE user_enterprises
    ADD CONSTRAINT user_enterprises_role_check
    CHECK (role IN ('ADMIN', 'USER')) NOT VALID;

ALTER TABLE user_enterprises
    VALIDATE CONSTRAINT user_enterprises_role_check;

CREATE INDEX IF NOT EXISTS idx_user_enterprises_authorization
    ON user_enterprises (user_id, enterprise_id, role);
