BEGIN;
DROP TRIGGER IF EXISTS trg_cost_history_immutable ON item_standard_cost_history;
DROP FUNCTION IF EXISTS prevent_cost_history_mutation();
DROP TABLE IF EXISTS item_standard_cost_history;
DROP INDEX IF EXISTS idx_cost_overhead_rules_centro;
DROP INDEX IF EXISTS idx_cost_overhead_rules_vigentes;
DROP TABLE IF EXISTS cost_overhead_rules;
DROP TYPE IF EXISTS cost_overhead_method_enum;
DROP TYPE IF EXISTS cost_overhead_base_enum;
ALTER TABLE cost_rollup_log
    DROP COLUMN IF EXISTS parent_code,
    DROP COLUMN IF EXISTS quantity,
    DROP COLUMN IF EXISTS lower_level_cost,
    DROP COLUMN IF EXISTS subcontract_cost,
    DROP COLUMN IF EXISTS machine_cost,
    DROP COLUMN IF EXISTS setup_cost;
ALTER TABLE item_standard_costs
    DROP COLUMN IF EXISTS overhead_detail,
    DROP COLUMN IF EXISTS lot_size,
    DROP COLUMN IF EXISTS lower_level_cost,
    DROP COLUMN IF EXISTS own_level_cost,
    DROP COLUMN IF EXISTS subcontract_cost,
    DROP COLUMN IF EXISTS machine_cost,
    DROP COLUMN IF EXISTS setup_cost;
COMMIT;
