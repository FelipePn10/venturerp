ALTER TABLE fiscal_exit_items DROP COLUMN address, DROP COLUMN lot, DROP COLUMN mask;
DROP TABLE IF EXISTS stock_parameters;
DROP TABLE IF EXISTS stock_lot_status_events;
ALTER TABLE stock_lots DROP CONSTRAINT IF EXISTS fk_stock_lots_status, DROP COLUMN grade, DROP COLUMN status_code;
DROP TABLE IF EXISTS stock_lot_statuses;
ALTER TABLE manufacturing_stock_item_controls DROP COLUMN lot_issue_rule;
