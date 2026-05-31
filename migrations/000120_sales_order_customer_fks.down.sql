ALTER TABLE sales_orders
    DROP CONSTRAINT IF EXISTS fk_so_bearer,
    DROP CONSTRAINT IF EXISTS fk_so_payment_term,
    DROP CONSTRAINT IF EXISTS fk_so_price_table,
    DROP CONSTRAINT IF EXISTS fk_so_tax_type;
