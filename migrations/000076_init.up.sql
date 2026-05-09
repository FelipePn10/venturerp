ALTER TABLE mrp_calculation_logs
RENAME COLUMN plan_id TO plan_code;

ALTER TABLE mrp_item_profiles
RENAME COLUMN plan_id TO plan_code;