BEGIN;
ALTER TABLE cutting_settings ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
UPDATE cutting_settings
SET enterprise_id=(SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL
  AND (SELECT COUNT(*) FROM enterprise)=1;
-- The legacy singleton has no reliable owner when there is no enterprise or
-- more than one tenant. Dropping it is safer than exposing one tenant's
-- defaults to another; the API supplies defaults when no settings row exists.
DELETE FROM cutting_settings WHERE enterprise_id IS NULL;
ALTER TABLE cutting_settings ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cutting_settings DROP CONSTRAINT IF EXISTS cutting_settings_pkey;
ALTER TABLE cutting_settings ADD PRIMARY KEY(enterprise_id);
DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conname='groups_enterprise_code_key') THEN ALTER TABLE groups ADD CONSTRAINT groups_enterprise_code_key UNIQUE(enterprise_id,code); END IF; END $$;
ALTER TABLE modifier ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
UPDATE modifier m SET enterprise_id=ue.enterprise_id FROM user_enterprises ue WHERE m.enterprise_id IS NULL AND m.created_by=ue.user_id AND (SELECT COUNT(*) FROM user_enterprises x WHERE x.user_id=m.created_by)=1;
UPDATE modifier SET enterprise_id=(SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise)=1;
DO $$ BEGIN IF EXISTS(SELECT 1 FROM modifier WHERE enterprise_id IS NULL) THEN RAISE EXCEPTION 'Nao foi possivel determinar a empresa de todos os modificadores PDM'; END IF; END $$;
ALTER TABLE modifier ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX idx_modifier_enterprise ON modifier(enterprise_id,id);
COMMIT;
