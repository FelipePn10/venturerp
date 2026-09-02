BEGIN;

ALTER TABLE purchase_order_items
    ADD COLUMN IF NOT EXISTS warehouse_id BIGINT,
    ADD COLUMN IF NOT EXISTS planned_order_code BIGINT,
    ADD COLUMN IF NOT EXISTS demand_type VARCHAR(30),
    ADD COLUMN IF NOT EXISTS demand_code BIGINT,
    ADD COLUMN IF NOT EXISTS sales_order_code BIGINT,
    ADD COLUMN IF NOT EXISTS production_order_id BIGINT,
    ADD COLUMN IF NOT EXISTS purchase_requisition_code BIGINT,
    ADD COLUMN IF NOT EXISTS purchase_requisition_item_id BIGINT;

CREATE INDEX IF NOT EXISTS ix_purchase_order_items_origin_trace
ON purchase_order_items(planned_order_code, demand_type, demand_code, sales_order_code, production_order_id);
CREATE INDEX IF NOT EXISTS ix_purchase_order_items_warehouse
ON purchase_order_items(warehouse_id, item_code);

CREATE UNIQUE INDEX IF NOT EXISTS ux_purchase_tolerances_idempotent
ON purchase_order_tolerances
(enterprise_id, tolerance_type, applies_to, interval_min, supplier_code) NULLS NOT DISTINCT;

CREATE TABLE IF NOT EXISTS procurement_immutable_audit (
    id BIGSERIAL PRIMARY KEY,
    enterprise_key BIGINT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_key TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('INSERT','UPDATE','DELETE')),
    before_state JSONB,
    after_state JSONB,
    actor_id UUID,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS ix_procurement_audit_tenant_entity
ON procurement_immutable_audit(enterprise_key, entity_type, entity_key, occurred_at DESC);

CREATE OR REPLACE FUNCTION record_procurement_immutable_audit() RETURNS TRIGGER AS $$
DECLARE
    old_row JSONB := CASE WHEN TG_OP = 'INSERT' THEN NULL ELSE to_jsonb(OLD) END;
    new_row JSONB := CASE WHEN TG_OP = 'DELETE' THEN NULL ELSE to_jsonb(NEW) END;
    row_data JSONB := COALESCE(new_row, old_row);
    tenant BIGINT;
    actor UUID;
BEGIN
    tenant := COALESCE((row_data->>'enterprise_id')::BIGINT, (row_data->>'enterprise_code')::BIGINT);
    IF tenant IS NULL AND TG_TABLE_NAME = 'purchase_order_items' THEN
        SELECT enterprise_code INTO tenant
        FROM purchase_orders
        WHERE code = (row_data->>'purchase_order_code')::BIGINT;
    END IF;
    actor := COALESCE(NULLIF(row_data->>'updated_by','')::UUID, NULLIF(row_data->>'created_by','')::UUID);
    INSERT INTO procurement_immutable_audit
        (enterprise_key, entity_type, entity_key, action, before_state, after_state, actor_id)
    VALUES
        (tenant, TG_TABLE_NAME, COALESCE(row_data->>'id', row_data->>'code', row_data->>'order_number', row_data->>'enterprise_code', row_data->>'enterprise_id'),
         TG_OP, old_row, new_row, actor);
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION prevent_procurement_audit_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'auditoria de suprimentos é imutável';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_procurement_audit_immutable ON procurement_immutable_audit;
CREATE TRIGGER trg_procurement_audit_immutable
BEFORE UPDATE OR DELETE ON procurement_immutable_audit
FOR EACH ROW EXECUTE FUNCTION prevent_procurement_audit_mutation();

DO $$
DECLARE table_name TEXT;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'purchase_orders','purchase_order_items','purchase_order_tolerances',
        'purchase_approval_limits','supplier_parameters','supplier_edi_messages',
        'procurement_parameters','third_party_service_prices','third_party_service_orders'
    ] LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS trg_%s_immutable_audit ON %I', table_name, table_name);
        EXECUTE format('CREATE TRIGGER trg_%s_immutable_audit AFTER INSERT OR UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION record_procurement_immutable_audit()', table_name, table_name);
    END LOOP;
END $$;

COMMIT;
