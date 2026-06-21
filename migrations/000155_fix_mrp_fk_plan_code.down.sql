ALTER TABLE mrp_item_profiles
    DROP CONSTRAINT IF EXISTS mrp_item_profiles_plan_code_fkey;

ALTER TABLE mrp_item_profiles
    ADD CONSTRAINT mrp_item_profiles_plan_id_fkey
        FOREIGN KEY (plan_code) REFERENCES production_plans(id) ON DELETE CASCADE;

ALTER TABLE mrp_calculation_logs
    DROP CONSTRAINT IF EXISTS mrp_calculation_logs_plan_code_fkey;

ALTER TABLE mrp_calculation_logs
    ADD CONSTRAINT mrp_calculation_logs_plan_id_fkey
        FOREIGN KEY (plan_code) REFERENCES production_plans(id) ON DELETE CASCADE;
