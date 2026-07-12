DROP INDEX IF EXISTS uq_stock_balances_tenant_item_warehouse;

ALTER TABLE stock_balances
    ADD CONSTRAINT stock_balances_item_code_mask_warehouse_id_key
    UNIQUE (item_code, mask, warehouse_id);
