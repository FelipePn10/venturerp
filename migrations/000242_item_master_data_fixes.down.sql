DROP INDEX IF EXISTS idx_fiscal_classifications_item_code;

ALTER TABLE fiscal_classifications
    DROP CONSTRAINT IF EXISTS fk_fiscal_classifications_item,
    DROP COLUMN IF EXISTS item_code;

ALTER TABLE items
    DROP CONSTRAINT IF EXISTS chk_items_supplies_warehouse_code_positive,
    DROP CONSTRAINT IF EXISTS chk_items_commercial_warranty_days_nonnegative,
    DROP CONSTRAINT IF EXISTS chk_items_planning_lots_nonnegative,
    DROP CONSTRAINT IF EXISTS chk_items_planning_abc_class,
    DROP COLUMN IF EXISTS accounting_calculate_pis_cofins,
    DROP COLUMN IF EXISTS accounting_active,
    DROP COLUMN IF EXISTS commercial_warranty_days,
    DROP COLUMN IF EXISTS supplies_harvest,
    DROP COLUMN IF EXISTS supplies_receiving_checklist,
    DROP COLUMN IF EXISTS supplies_warehouse_code,
    DROP COLUMN IF EXISTS supplies_purchase_uom,
    DROP COLUMN IF EXISTS planning_safety_stock,
    DROP COLUMN IF EXISTS planning_active,
    DROP COLUMN IF EXISTS planning_exclusive,
    DROP COLUMN IF EXISTS planning_critical,
    DROP COLUMN IF EXISTS planning_multiple_lot,
    DROP COLUMN IF EXISTS planning_minimum_lot,
    DROP COLUMN IF EXISTS planning_abc_class,
    DROP COLUMN IF EXISTS name;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM items WHERE warehouse_unit_of_measurement::text IN ('L', 'CX', 'PC', 'GL', 'PAR'))
       OR EXISTS (SELECT 1 FROM item_structures WHERE unit_of_measurement::text IN ('L', 'CX', 'PC', 'GL', 'PAR')) THEN
        RAISE EXCEPTION 'cannot remove new units of measurement while they are in use';
    END IF;
END $$;

ALTER TABLE item_structures ALTER COLUMN unit_of_measurement DROP DEFAULT;
ALTER TABLE items ALTER COLUMN warehouse_unit_of_measurement DROP DEFAULT;
ALTER TYPE unit_of_measurement_enum RENAME TO unit_of_measurement_enum_extended;
CREATE TYPE unit_of_measurement_enum AS ENUM (
    'MM', 'CM', 'M', 'IN', 'KG', 'M2', 'M3', 'UN', 'MICROMETRO', 'TONELADA'
);
ALTER TABLE item_structures
    ALTER COLUMN unit_of_measurement TYPE unit_of_measurement_enum
    USING unit_of_measurement::text::unit_of_measurement_enum;
ALTER TABLE item_structures ALTER COLUMN unit_of_measurement SET DEFAULT 'UN';
ALTER TABLE items
    ALTER COLUMN warehouse_unit_of_measurement TYPE unit_of_measurement_enum
    USING warehouse_unit_of_measurement::text::unit_of_measurement_enum;
ALTER TABLE items ALTER COLUMN warehouse_unit_of_measurement SET DEFAULT 'UN';
DROP TYPE unit_of_measurement_enum_extended;
