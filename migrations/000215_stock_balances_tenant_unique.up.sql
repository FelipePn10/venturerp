ALTER TABLE stock_balances
    DROP CONSTRAINT IF EXISTS stock_balances_item_code_mask_warehouse_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_balances_tenant_item_warehouse
    ON stock_balances (enterprise_id, item_code, mask, warehouse_id)
    WHERE enterprise_id IS NOT NULL;
