BEGIN;
DROP INDEX IF EXISTS idx_production_orders_tenant_number;
ALTER TABLE manufacturing_stock_item_controls
    DROP CONSTRAINT IF EXISTS manufacturing_transfer_line_chk,
    DROP COLUMN IF EXISTS line_warehouse_id,
    DROP COLUMN IF EXISTS automatic_issue_type;
COMMIT;
