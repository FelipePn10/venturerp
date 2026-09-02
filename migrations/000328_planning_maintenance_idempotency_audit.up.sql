CREATE TABLE IF NOT EXISTS http_idempotency_records (
    scope_key TEXT PRIMARY KEY,
    request_fingerprint TEXT NOT NULL,
    status_code INT,
    response_body BYTEA,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_http_idempotency_expiry ON http_idempotency_records(expires_at);

CREATE TABLE IF NOT EXISTS operational_mutation_audit (
    id BIGSERIAL PRIMARY KEY,
    table_name TEXT NOT NULL,
    operation TEXT NOT NULL CHECK (operation IN ('INSERT','UPDATE','DELETE')),
    row_key TEXT NOT NULL,
    before_data JSONB,
    after_data JSONB,
    enterprise_id BIGINT,
    actor_id UUID,
    idempotency_key TEXT,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_operational_mutation_audit_lookup
    ON operational_mutation_audit(table_name,row_key,occurred_at DESC);

CREATE OR REPLACE FUNCTION record_operational_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    old_data JSONB;
    new_data JSONB;
    key_value TEXT;
    tenant_value BIGINT;
    actor_value UUID;
BEGIN
    old_data := CASE WHEN TG_OP IN ('UPDATE','DELETE') THEN to_jsonb(OLD) ELSE NULL END;
    new_data := CASE WHEN TG_OP IN ('INSERT','UPDATE') THEN to_jsonb(NEW) ELSE NULL END;
    key_value := COALESCE(new_data->>'id',new_data->>'code',old_data->>'id',old_data->>'code','unknown');
    tenant_value := NULLIF(COALESCE(new_data->>'enterprise_id',new_data->>'enterprise_code',old_data->>'enterprise_id',old_data->>'enterprise_code'),'')::BIGINT;
    IF tenant_value IS NULL AND TG_TABLE_NAME='maintenance_plans' THEN
        SELECT enterprise_id INTO tenant_value FROM machines
        WHERE id=NULLIF(COALESCE(new_data->>'machine_id',old_data->>'machine_id'),'')::BIGINT;
    ELSIF tenant_value IS NULL AND TG_TABLE_NAME='maintenance_orders' THEN
        SELECT m.enterprise_id INTO tenant_value
        FROM maintenance_plans p JOIN machines m ON m.id=p.machine_id
        WHERE p.id=NULLIF(COALESCE(new_data->>'plan_id',old_data->>'plan_id'),'')::BIGINT;
    END IF;
    actor_value := NULLIF(COALESCE(new_data->>'updated_by',new_data->>'created_by',old_data->>'updated_by',old_data->>'created_by'),'')::UUID;
    INSERT INTO operational_mutation_audit(table_name,operation,row_key,before_data,after_data,enterprise_id,actor_id)
    VALUES(TG_TABLE_NAME,TG_OP,key_value,old_data,new_data,tenant_value,actor_value);
    RETURN CASE WHEN TG_OP='DELETE' THEN OLD ELSE NEW END;
END $$;

CREATE OR REPLACE FUNCTION protect_operational_mutation_audit() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'operational_mutation_audit is append-only';
END $$;

DROP TRIGGER IF EXISTS trg_operational_mutation_audit_immutable ON operational_mutation_audit;
CREATE TRIGGER trg_operational_mutation_audit_immutable
BEFORE UPDATE OR DELETE ON operational_mutation_audit
FOR EACH ROW EXECUTE FUNCTION protect_operational_mutation_audit();

DROP TRIGGER IF EXISTS trg_audit_log_immutable ON audit_log;
CREATE TRIGGER trg_audit_log_immutable BEFORE UPDATE OR DELETE ON audit_log
FOR EACH ROW EXECUTE FUNCTION protect_operational_mutation_audit();

DO $$
DECLARE table_to_audit TEXT;
BEGIN
    FOREACH table_to_audit IN ARRAY ARRAY[
        'maintenance_plans','maintenance_orders','machine_downtimes',
        'planning_params','mrp_calculation_logs','configured_item_rules',
        'shipment_loads','shipment_load_shipments','sales_forecasts',
        'mrp_item_profiles','mrp_planned_suggestions','capacity_requirements','production_sequences'
    ] LOOP
        IF to_regclass('public.'||table_to_audit) IS NOT NULL THEN
            EXECUTE format('DROP TRIGGER IF EXISTS trg_operational_audit ON public.%I',table_to_audit);
            EXECUTE format('CREATE TRIGGER trg_operational_audit AFTER INSERT OR UPDATE OR DELETE ON public.%I FOR EACH ROW EXECUTE FUNCTION record_operational_mutation()',table_to_audit);
        END IF;
    END LOOP;
END $$;
