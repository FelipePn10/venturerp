BEGIN;

DO $$
DECLARE table_name TEXT;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'purchase_orders','purchase_order_items','purchase_order_tolerances',
        'purchase_approval_limits','supplier_parameters','supplier_edi_messages',
        'procurement_parameters','third_party_service_prices','third_party_service_orders'
    ] LOOP
        IF to_regclass(table_name) IS NOT NULL THEN
            EXECUTE format('DROP TRIGGER IF EXISTS trg_%s_immutable_audit ON %I', table_name, table_name);
        END IF;
    END LOOP;
END $$;

DROP FUNCTION IF EXISTS record_procurement_immutable_audit();
DROP TRIGGER IF EXISTS trg_procurement_audit_immutable ON procurement_immutable_audit;
DROP FUNCTION IF EXISTS prevent_procurement_audit_mutation();
DROP TABLE IF EXISTS procurement_immutable_audit;
DROP INDEX IF EXISTS ux_purchase_tolerances_idempotent;
DROP INDEX IF EXISTS ix_purchase_order_items_origin_trace;
DROP INDEX IF EXISTS ix_purchase_order_items_warehouse;
ALTER TABLE purchase_order_items
    DROP COLUMN IF EXISTS purchase_requisition_item_id,
    DROP COLUMN IF EXISTS purchase_requisition_code,
    DROP COLUMN IF EXISTS production_order_id,
    DROP COLUMN IF EXISTS sales_order_code,
    DROP COLUMN IF EXISTS demand_code,
    DROP COLUMN IF EXISTS demand_type,
    DROP COLUMN IF EXISTS planned_order_code,
    DROP COLUMN IF EXISTS warehouse_id;

COMMIT;
