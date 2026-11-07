ALTER TABLE modifier DROP COLUMN is_active;
ALTER TABLE groups DROP COLUMN is_active;
ALTER TABLE items DROP CONSTRAINT fk_items_accounting_inventory_group;
ALTER TABLE groups ADD CONSTRAINT groups_code_key UNIQUE (code);
ALTER TABLE items ADD CONSTRAINT fk_items_accounting_inventory_group
    FOREIGN KEY (accounting_inventory_group_code) REFERENCES groups (code);
DROP INDEX IF EXISTS idx_tools_item;
ALTER TABLE tools DROP COLUMN item_code;
DROP INDEX IF EXISTS idx_items_exclusive_customer;
ALTER TABLE items
    DROP CONSTRAINT chk_items_cutting_grain,
    DROP CONSTRAINT chk_items_volumes_per_unit,
    DROP COLUMN is_scrap, DROP COLUMN engineering_requires_drawing, DROP COLUMN engineering_cutting_grain,
    DROP COLUMN engineering_cutting_no_rotation, DROP COLUMN supplies_requires_certificate,
    DROP COLUMN commercial_exclusive_customer_code, DROP COLUMN commercial_volumes_per_unit;
SELECT instalar_conferencia_de_empresa();
