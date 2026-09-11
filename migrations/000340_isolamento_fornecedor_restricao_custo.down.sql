DROP INDEX IF EXISTS idx_suppliers_enterprise;
DROP INDEX IF EXISTS idx_supplier_contacts_enterprise;
DROP INDEX IF EXISTS idx_supplier_contact_phones_enterprise;
DROP INDEX IF EXISTS idx_supplier_addresses_enterprise;
DROP INDEX IF EXISTS idx_restriction_reasons_enterprise;
DROP INDEX IF EXISTS idx_work_center_costs_enterprise;

ALTER TABLE suppliers               DROP CONSTRAINT IF EXISTS fk_suppliers_enterprise;
ALTER TABLE restriction_reasons     DROP CONSTRAINT IF EXISTS fk_restriction_reasons_enterprise;
ALTER TABLE work_center_costs       DROP CONSTRAINT IF EXISTS fk_work_center_costs_enterprise;

ALTER TABLE suppliers               DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE restriction_reasons     DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE work_center_costs       DROP COLUMN IF EXISTS enterprise_id;

ALTER TABLE restriction_reasons DROP CONSTRAINT IF EXISTS restriction_reasons_enterprise_code_key;
ALTER TABLE work_center_costs   DROP CONSTRAINT IF EXISTS work_center_costs_enterprise_wc_key;

ALTER TABLE restriction_reasons ADD CONSTRAINT restriction_reasons_code_key          UNIQUE (code);
ALTER TABLE work_center_costs   ADD CONSTRAINT work_center_costs_work_center_id_key  UNIQUE (work_center_id);
