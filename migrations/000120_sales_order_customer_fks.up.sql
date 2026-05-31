-- Connect sales_orders loose BIGINT codes to the customer module tables
ALTER TABLE sales_orders
    ADD CONSTRAINT fk_so_tax_type
        FOREIGN KEY (tax_type_code)    REFERENCES tax_types(code),
    ADD CONSTRAINT fk_so_price_table
        FOREIGN KEY (price_table_code) REFERENCES sales_tables(code),
    ADD CONSTRAINT fk_so_payment_term
        FOREIGN KEY (payment_term_code) REFERENCES payment_conditions(code),
    ADD CONSTRAINT fk_so_bearer
        FOREIGN KEY (bearer_code)      REFERENCES carriers(code);
