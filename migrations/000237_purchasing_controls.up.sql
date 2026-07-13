BEGIN;

-- Tenant retrofit for the purchasing registers introduced before tenant scoping.
ALTER TABLE purchase_price_tables
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id),
    ADD COLUMN IF NOT EXISTS supplier_code BIGINT;

UPDATE purchase_price_tables p
SET enterprise_id = ue.enterprise_id
FROM user_enterprises ue
WHERE p.enterprise_id IS NULL
  AND p.created_by = ue.user_id
  AND (SELECT COUNT(*) FROM user_enterprises x WHERE x.user_id = p.created_by) = 1;

UPDATE purchase_price_tables
SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;

ALTER TABLE purchase_price_tables DROP CONSTRAINT IF EXISTS purchase_price_tables_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS ux_purchase_price_tables_tenant_code
    ON purchase_price_tables(enterprise_id, code) WHERE enterprise_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS ix_purchase_price_tables_supplier
    ON purchase_price_tables(enterprise_id, supplier_code) WHERE is_active;

ALTER TABLE purchase_price_table_items
    ADD COLUMN IF NOT EXISTS update_replacement_value BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE public.purchase_order_items ALTER COLUMN unit_price TYPE NUMERIC(18,6);
ALTER TABLE public.fiscal_entry_items ALTER COLUMN unit_price TYPE NUMERIC(18,6);

CREATE TABLE IF NOT EXISTS purchase_price_item_adjustments (
    id BIGSERIAL PRIMARY KEY,
    price_item_id BIGINT NOT NULL REFERENCES purchase_price_table_items(id) ON DELETE CASCADE,
    sequence INT NOT NULL DEFAULT 1,
    adjustment_kind VARCHAR(10) NOT NULL CHECK (adjustment_kind IN ('DISCOUNT','SURCHARGE')),
    calculation_type VARCHAR(10) NOT NULL CHECK (calculation_type IN ('PERCENT','FIXED')),
    value NUMERIC(18,6) NOT NULL CHECK (value >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(price_item_id, sequence)
);

ALTER TABLE item_preferred_suppliers
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id),
    ADD COLUMN IF NOT EXISTS mask VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS xml_uom VARCHAR(10),
    ADD COLUMN IF NOT EXISTS conversion_factor NUMERIC(18,8),
    ADD COLUMN IF NOT EXISTS package_quantity NUMERIC(18,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS is_preferred BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS supplier_uf VARCHAR(2),
    ADD COLUMN IF NOT EXISTS classification_id BIGINT REFERENCES item_classifications(id),
    ADD COLUMN IF NOT EXISTS classification_date DATE,
    ADD COLUMN IF NOT EXISTS classification_grade NUMERIC(9,4),
    ADD COLUMN IF NOT EXISTS direct_billing BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS third_party_order BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS ignore_avg_cost_addition BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS ecommerce BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS barcode VARCHAR(80),
    ADD COLUMN IF NOT EXISTS notes TEXT,
    ADD COLUMN IF NOT EXISTS valid_until DATE,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE item_preferred_suppliers s
SET enterprise_id = ue.enterprise_id
FROM user_enterprises ue
WHERE s.enterprise_id IS NULL
  AND s.created_by = ue.user_id
  AND (SELECT COUNT(*) FROM user_enterprises x WHERE x.user_id = s.created_by) = 1;

UPDATE item_preferred_suppliers
SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;

UPDATE item_preferred_suppliers SET is_preferred = TRUE WHERE ranking = 1;
ALTER TABLE item_preferred_suppliers
    DROP CONSTRAINT IF EXISTS item_preferred_suppliers_item_code_supplier_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS ux_item_suppliers_tenant_occurrence
    ON item_preferred_suppliers(enterprise_id, item_code, supplier_code, mask)
    WHERE enterprise_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS ux_item_suppliers_barcode
    ON item_preferred_suppliers(enterprise_id, supplier_code, barcode)
    WHERE barcode IS NOT NULL AND btrim(barcode) <> '' AND is_active;
CREATE INDEX IF NOT EXISTS ix_item_suppliers_supplier
    ON item_preferred_suppliers(enterprise_id, supplier_code, item_code) WHERE is_active;

CREATE TABLE IF NOT EXISTS item_supplier_quality_reports (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    item_supplier_id BIGINT NOT NULL REFERENCES item_preferred_suppliers(id) ON DELETE CASCADE,
    registered_on DATE NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING','APPROVED','REJECTED','EXPIRED')),
    report_file_name TEXT,
    report_content_type TEXT,
    report_content BYTEA,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL
);
CREATE INDEX IF NOT EXISTS ix_item_supplier_quality
    ON item_supplier_quality_reports(enterprise_id, item_supplier_id, registered_on DESC);

-- A fiscal entry needs an explicit tenant and line UOM for safe price-table imports.
ALTER TABLE public.fiscal_entries
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE public.fiscal_entry_items
    ADD COLUMN IF NOT EXISTS uom VARCHAR(10);

UPDATE fiscal_entries f
SET enterprise_id = e.id
FROM purchase_orders po
JOIN enterprise e ON e.code = po.enterprise_code
WHERE f.enterprise_id IS NULL AND f.purchase_order_code = po.code;

UPDATE fiscal_entries f
SET enterprise_id = ue.enterprise_id
FROM user_enterprises ue
WHERE f.enterprise_id IS NULL
  AND f.created_by = ue.user_id
  AND (SELECT COUNT(*) FROM user_enterprises x WHERE x.user_id = f.created_by) = 1;

UPDATE fiscal_entries
SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;
CREATE INDEX IF NOT EXISTS ix_fiscal_entries_tenant_period
    ON fiscal_entries(enterprise_id, supplier_code, data_entrada) WHERE is_active;

CREATE TABLE IF NOT EXISTS purchase_order_tolerances (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    tolerance_type VARCHAR(20) NOT NULL CHECK (tolerance_type IN ('QUANTITY','ITEM_PRICE','PRODUCTS_TOTAL')),
    applies_to VARCHAR(20) NOT NULL CHECK (applies_to IN ('ENTRY_INVOICE','RECEIVING_NOTICE','ALL')),
    interval_min NUMERIC(18,6) NOT NULL DEFAULT 0,
    interval_max NUMERIC(18,6),
    tolerance_value NUMERIC(18,6) NOT NULL CHECK (tolerance_value >= 0),
    value_type VARCHAR(10) NOT NULL CHECK (value_type IN ('PERCENT','FIXED')),
    supplier_code BIGINT,
    action VARCHAR(10) NOT NULL CHECK (action IN ('BLOCK','WARN')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    CHECK (interval_max IS NULL OR interval_max >= interval_min)
);
CREATE INDEX IF NOT EXISTS ix_purchase_tolerances_resolution
    ON purchase_order_tolerances(enterprise_id, applies_to, tolerance_type, supplier_code, interval_min)
    WHERE is_active;

COMMIT;
