BEGIN;
DROP TABLE IF EXISTS machine_downtimes;
DROP TABLE IF EXISTS machine_special_values;
DROP TABLE IF EXISTS machine_special_fields;
DROP TABLE IF EXISTS machine_service_responsibles;
DROP TABLE IF EXISTS machine_service_items;
DROP TABLE IF EXISTS machine_preventive_services;
DROP TABLE IF EXISTS preventive_services;
ALTER TABLE machines DROP COLUMN IF EXISTS maintenance_responsible_employee_id, DROP COLUMN IF EXISTS is_preferred,
 DROP COLUMN IF EXISTS brand, DROP COLUMN IF EXISTS supplier_code, DROP COLUMN IF EXISTS preparation_time_unit,
 DROP COLUMN IF EXISTS preparation_time, DROP COLUMN IF EXISTS acquired_on, DROP COLUMN IF EXISTS usage_description;
DROP TABLE IF EXISTS employee_credit_limits;
DROP INDEX IF EXISTS uq_employee_cost_center_manager;
DROP INDEX IF EXISTS uq_employee_cost_center_supervisor;
DROP TABLE IF EXISTS employee_functions;
DROP TABLE IF EXISTS employee_contacts;
DROP INDEX IF EXISTS uq_employees_tenant_code;
ALTER TABLE employees DROP COLUMN IF EXISTS enterprise_id;
COMMIT;
