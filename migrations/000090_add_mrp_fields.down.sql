BEGIN;

DROP TABLE IF EXISTS mps_schedule;
DROP TABLE IF EXISTS kanban_cards;

ALTER TABLE planned_orders
  DROP COLUMN IF EXISTS coverage_days;

ALTER TABLE planned_orders
  DROP COLUMN IF EXISTS safety_time_days;

ALTER TABLE stock_snapshots
  DROP COLUMN IF EXISTS min_max_active;

ALTER TABLE stock_snapshots
  DROP COLUMN IF EXISTS maximum_stock;

COMMIT;
