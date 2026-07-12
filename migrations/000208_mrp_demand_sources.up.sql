CREATE TABLE IF NOT EXISTS item_classification_assignments (
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    item_code BIGINT NOT NULL,
    classification_id BIGINT NOT NULL REFERENCES item_classifications(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_id, item_code, classification_id)
);

CREATE INDEX IF NOT EXISTS idx_item_classification_assignments_lookup
    ON item_classification_assignments (enterprise_id, classification_id, item_code);
CREATE INDEX IF NOT EXISTS idx_sales_orders_mrp
    ON sales_orders (enterprise_code, is_active, is_blocked, status);
CREATE INDEX IF NOT EXISTS idx_sales_order_items_mrp
    ON sales_order_items (sales_order_code, is_active, status, delivery_date);
