BEGIN;

ALTER TABLE supplier_enterprises DROP CONSTRAINT IF EXISTS fk_supplier_enterprises_price_table;
DROP TABLE IF EXISTS purchase_price_table_items;
DROP TABLE IF EXISTS purchase_price_tables;

COMMIT;
