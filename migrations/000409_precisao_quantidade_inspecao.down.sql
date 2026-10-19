ALTER TABLE procurement_inspection_dispositions
    ALTER COLUMN approved_qty TYPE NUMERIC(15,4),
    ALTER COLUMN rejected_qty TYPE NUMERIC(15,4);
ALTER TABLE procurement_records
    ALTER COLUMN quantity TYPE NUMERIC(15,4);
ALTER TABLE receiving_inspection_analyses
    ALTER COLUMN conform_qty    TYPE NUMERIC(15,4),
    ALTER COLUMN rejected_qty   TYPE NUMERIC(15,4),
    ALTER COLUMN rework_qty     TYPE NUMERIC(15,4),
    ALTER COLUMN restricted_qty TYPE NUMERIC(15,4);
ALTER TABLE receiving_inspection_orders
    ALTER COLUMN quantity       TYPE NUMERIC(15,4),
    ALTER COLUMN inspected_qty  TYPE NUMERIC(15,4),
    ALTER COLUMN approved_qty   TYPE NUMERIC(15,4),
    ALTER COLUMN rejected_qty   TYPE NUMERIC(15,4),
    ALTER COLUMN rework_qty     TYPE NUMERIC(15,4),
    ALTER COLUMN restricted_qty TYPE NUMERIC(15,4);
