DROP INDEX IF EXISTS uq_stock_cycle_count_open_scope;

ALTER TABLE stock_cycle_counts
    DROP COLUMN IF EXISTS policy_days,
    DROP COLUMN IF EXISTS origin;

DROP TRIGGER IF EXISTS trg_item_cycle_count_policy_activation ON items;
DROP FUNCTION IF EXISTS set_item_cycle_count_policy_activation();

ALTER TABLE items
    DROP COLUMN IF EXISTS cyclical_count_policy_activated_at;
