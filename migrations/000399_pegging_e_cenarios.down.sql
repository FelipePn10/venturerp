DROP TABLE IF EXISTS planning_scenario_changes;
DROP TABLE IF EXISTS planning_scenarios;
ALTER TABLE production_plans DROP COLUMN IF EXISTS is_scenario;
DROP TABLE IF EXISTS mrp_pegging;
