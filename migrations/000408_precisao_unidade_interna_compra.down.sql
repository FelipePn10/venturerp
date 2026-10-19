ALTER TABLE purchase_order_items
    ALTER COLUMN internal_qty   TYPE NUMERIC(15,4),
    ALTER COLUMN internal_price TYPE NUMERIC(15,4);
