BEGIN;
ALTER TABLE manufacturing_stock_item_controls
    ADD COLUMN automatic_issue_type VARCHAR(12) NOT NULL DEFAULT 'ISSUE'
        CHECK (automatic_issue_type IN ('ISSUE','TRANSFER')),
    ADD COLUMN line_warehouse_id BIGINT,
    ADD CONSTRAINT manufacturing_transfer_line_chk
        CHECK (automatic_issue_type <> 'TRANSFER' OR line_warehouse_id IS NOT NULL);
CREATE UNIQUE INDEX idx_production_orders_tenant_number
    ON production_orders(enterprise_id,order_number);
COMMIT;
