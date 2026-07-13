BEGIN;

DROP TABLE IF EXISTS purchase_order_tolerances;
DROP TABLE IF EXISTS item_supplier_quality_reports;
DROP TABLE IF EXISTS purchase_price_item_adjustments;

DROP INDEX IF EXISTS ix_fiscal_entries_tenant_period;
ALTER TABLE public.fiscal_entry_items DROP COLUMN IF EXISTS uom;
ALTER TABLE public.fiscal_entry_items ALTER COLUMN unit_price TYPE NUMERIC(15,2);
ALTER TABLE public.fiscal_entries DROP COLUMN IF EXISTS enterprise_id;

DROP INDEX IF EXISTS ux_item_suppliers_barcode;
DROP INDEX IF EXISTS ux_item_suppliers_tenant_occurrence;
DROP INDEX IF EXISTS ix_item_suppliers_supplier;
DELETE FROM item_preferred_suppliers newer
USING item_preferred_suppliers older
WHERE newer.item_code = older.item_code
  AND newer.supplier_code = older.supplier_code
  AND newer.id > older.id;
ALTER TABLE item_preferred_suppliers
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS valid_until,
    DROP COLUMN IF EXISTS notes,
    DROP COLUMN IF EXISTS barcode,
    DROP COLUMN IF EXISTS ecommerce,
    DROP COLUMN IF EXISTS ignore_avg_cost_addition,
    DROP COLUMN IF EXISTS third_party_order,
    DROP COLUMN IF EXISTS direct_billing,
    DROP COLUMN IF EXISTS classification_grade,
    DROP COLUMN IF EXISTS classification_date,
    DROP COLUMN IF EXISTS classification_id,
    DROP COLUMN IF EXISTS supplier_uf,
    DROP COLUMN IF EXISTS is_preferred,
    DROP COLUMN IF EXISTS package_quantity,
    DROP COLUMN IF EXISTS conversion_factor,
    DROP COLUMN IF EXISTS xml_uom,
    DROP COLUMN IF EXISTS mask,
    DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE item_preferred_suppliers
    ADD CONSTRAINT item_preferred_suppliers_item_code_supplier_code_key UNIQUE(item_code, supplier_code);

ALTER TABLE purchase_price_table_items
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS update_replacement_value;
ALTER TABLE public.purchase_order_items ALTER COLUMN unit_price TYPE NUMERIC(15,4);

DROP INDEX IF EXISTS ix_purchase_price_tables_supplier;
DROP INDEX IF EXISTS ux_purchase_price_tables_tenant_code;
WITH duplicates AS (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY code ORDER BY id) AS occurrence
    FROM purchase_price_tables
), next_codes AS (
    SELECT id, (SELECT COALESCE(MAX(code), 0) FROM purchase_price_tables)
               + ROW_NUMBER() OVER (ORDER BY id) AS new_code
    FROM duplicates WHERE occurrence > 1
)
UPDATE purchase_price_tables p SET code = n.new_code
FROM next_codes n WHERE p.id = n.id;
ALTER TABLE purchase_price_tables
    DROP COLUMN IF EXISTS supplier_code,
    DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE purchase_price_tables
    ADD CONSTRAINT purchase_price_tables_code_key UNIQUE(code);

COMMIT;
