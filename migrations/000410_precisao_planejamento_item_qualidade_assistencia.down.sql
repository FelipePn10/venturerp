ALTER TABLE technical_assistance_rma_items
    ALTER COLUMN quantity TYPE NUMERIC(15,4);
ALTER TABLE technical_assistance_call_items
    ALTER COLUMN quantity TYPE NUMERIC(15,4);

ALTER TABLE non_conformances
    ALTER COLUMN nonconform_qty TYPE NUMERIC(15,4);
ALTER TABLE quality_records
    ALTER COLUMN inspected_qty TYPE NUMERIC(15,4),
    ALTER COLUMN approved_qty  TYPE NUMERIC(15,4),
    ALTER COLUMN rejected_qty  TYPE NUMERIC(15,4);

-- Volta a inteiro arredondando: frações gravadas depois da subida se perdem.
ALTER TABLE items
    ALTER COLUMN warehouse_minimum_stock TYPE INTEGER USING ROUND(warehouse_minimum_stock)::INTEGER,
    ALTER COLUMN planning_minimum_lot    TYPE BIGINT  USING ROUND(planning_minimum_lot)::BIGINT,
    ALTER COLUMN planning_multiple_lot   TYPE BIGINT  USING ROUND(planning_multiple_lot)::BIGINT,
    ALTER COLUMN planning_safety_stock   TYPE BIGINT  USING ROUND(planning_safety_stock)::BIGINT;
