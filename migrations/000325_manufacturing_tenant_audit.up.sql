BEGIN;

ALTER TABLE operations ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE manufacturing_routes ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);

UPDATE manufacturing_routes route
	SET enterprise_id = candidate.enterprise_id
	FROM (
		SELECT code, MIN(enterprise_id) AS enterprise_id
		FROM items
		GROUP BY code
		HAVING COUNT(DISTINCT enterprise_id) = 1
	) candidate
	WHERE candidate.code = route.item_code AND route.enterprise_id IS NULL;

UPDATE operations operation
	SET enterprise_id = candidate.enterprise_id
	FROM (
		SELECT user_id, MIN(enterprise_id) AS enterprise_id
		FROM user_enterprises
		GROUP BY user_id
		HAVING COUNT(DISTINCT enterprise_id) = 1
	) candidate
	WHERE candidate.user_id = operation.created_by AND operation.enterprise_id IS NULL;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM operations WHERE enterprise_id IS NULL)
       OR EXISTS (SELECT 1 FROM manufacturing_routes WHERE enterprise_id IS NULL) THEN
        RAISE EXCEPTION 'Não foi possível determinar o tenant dos cadastros de manufatura';
    END IF;
END $$;

ALTER TABLE operations ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE manufacturing_routes ALTER COLUMN enterprise_id SET NOT NULL;

ALTER TABLE operations DROP CONSTRAINT IF EXISTS operations_code_key;
ALTER TABLE manufacturing_routes DROP CONSTRAINT IF EXISTS manufacturing_routes_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_operations_enterprise_code ON operations(enterprise_id, code);
CREATE UNIQUE INDEX IF NOT EXISTS uq_manufacturing_routes_enterprise_code ON manufacturing_routes(enterprise_id, code);
CREATE INDEX IF NOT EXISTS idx_operations_enterprise_active ON operations(enterprise_id, is_active, code);
CREATE INDEX IF NOT EXISTS idx_routes_enterprise_item ON manufacturing_routes(enterprise_id, item_code, is_active);

CREATE TABLE manufacturing_structural_audit (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    entity_type TEXT NOT NULL,
    entity_key TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('INSERT','UPDATE','DELETE')),
    before_state JSONB,
    after_state JSONB,
    actor_id UUID,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_manufacturing_structural_audit_tenant
    ON manufacturing_structural_audit(enterprise_id, entity_type, occurred_at DESC);

CREATE OR REPLACE FUNCTION record_manufacturing_structural_audit() RETURNS TRIGGER AS $$
DECLARE
    old_row JSONB := CASE WHEN TG_OP = 'INSERT' THEN NULL ELSE to_jsonb(OLD) END;
    new_row JSONB := CASE WHEN TG_OP = 'DELETE' THEN NULL ELSE to_jsonb(NEW) END;
    tenant_id BIGINT;
    key_value TEXT;
    actor UUID;
BEGIN
    IF TG_TABLE_NAME = 'groups' THEN
        tenant_id := COALESCE((new_row->>'enterprise_id')::BIGINT, (old_row->>'enterprise_id')::BIGINT);
        key_value := COALESCE(new_row->>'code', old_row->>'code');
    ELSIF TG_TABLE_NAME = 'item_structures' THEN
        SELECT enterprise_id INTO tenant_id FROM items
        WHERE code = COALESCE((new_row->>'parent_code')::BIGINT, (old_row->>'parent_code')::BIGINT);
        key_value := COALESCE(new_row->>'parent_code', old_row->>'parent_code') || '/' || COALESCE(new_row->>'child_code', old_row->>'child_code');
    ELSE
        tenant_id := COALESCE((new_row->>'enterprise_id')::BIGINT, (old_row->>'enterprise_id')::BIGINT);
        key_value := COALESCE(new_row->>'id', old_row->>'id');
    END IF;
    actor := NULLIF(COALESCE(new_row->>'created_by', old_row->>'created_by', ''), '')::UUID;
    INSERT INTO manufacturing_structural_audit(enterprise_id,entity_type,entity_key,action,before_state,after_state,actor_id)
    VALUES (tenant_id,TG_TABLE_NAME,key_value,TG_OP,old_row,new_row,actor);
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION prevent_manufacturing_audit_mutation() RETURNS TRIGGER AS $$
BEGIN RAISE EXCEPTION 'auditoria de manufatura é imutável'; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_manufacturing_audit_immutable
BEFORE UPDATE OR DELETE ON manufacturing_structural_audit
FOR EACH ROW EXECUTE FUNCTION prevent_manufacturing_audit_mutation();

DO $$ DECLARE table_name TEXT; BEGIN
    FOREACH table_name IN ARRAY ARRAY['groups','item_structures','operations','manufacturing_routes'] LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS trg_%s_structural_audit ON %I', table_name, table_name);
        EXECUTE format('CREATE TRIGGER trg_%s_structural_audit AFTER INSERT OR UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION record_manufacturing_structural_audit()', table_name, table_name);
    END LOOP;
END $$;

COMMIT;
