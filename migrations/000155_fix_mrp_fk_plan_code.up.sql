-- Fix mrp_calculation_logs and mrp_item_profiles FK constraints:
-- plan_code was incorrectly referencing production_plans.id (BIGSERIAL auto-id)
-- but the domain uses production_plans.code (user-defined business key)

ALTER TABLE mrp_calculation_logs
    DROP CONSTRAINT IF EXISTS mrp_calculation_logs_plan_id_fkey;

ALTER TABLE mrp_calculation_logs
    ADD CONSTRAINT mrp_calculation_logs_plan_code_fkey
        FOREIGN KEY (plan_code) REFERENCES production_plans(code) ON DELETE CASCADE;

ALTER TABLE mrp_item_profiles
    DROP CONSTRAINT IF EXISTS mrp_item_profiles_plan_id_fkey;

ALTER TABLE mrp_item_profiles
    ADD CONSTRAINT mrp_item_profiles_plan_code_fkey
        FOREIGN KEY (plan_code) REFERENCES production_plans(code) ON DELETE CASCADE;
