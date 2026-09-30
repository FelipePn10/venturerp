BEGIN;
DROP TRIGGER IF EXISTS trg_customer_material_remittances_audit ON customer_material_remittances;
DROP TRIGGER IF EXISTS trg_customer_material_items_audit ON customer_material_items;
DROP TRIGGER IF EXISTS trg_customer_material_movements_audit ON customer_material_movements;
DROP FUNCTION IF EXISTS record_customer_material_audit();
DROP TRIGGER IF EXISTS trg_customer_material_audit_immutable ON customer_material_audit;
DROP FUNCTION IF EXISTS prevent_customer_material_audit_mutation();
DROP TABLE IF EXISTS customer_material_audit;
COMMIT;
