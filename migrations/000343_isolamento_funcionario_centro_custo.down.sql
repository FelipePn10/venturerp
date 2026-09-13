DROP INDEX IF EXISTS idx_cost_centers_enterprise;
DROP INDEX IF EXISTS idx_employees_enterprise;
ALTER TABLE cost_centers DROP CONSTRAINT IF EXISTS fk_cost_centers_enterprise;
ALTER TABLE cost_centers DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE employees    ALTER COLUMN enterprise_id DROP NOT NULL;
