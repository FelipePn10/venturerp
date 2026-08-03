-- Compensatory, idempotent schema repair for databases that skipped 000242
-- because their recorded golang-migrate version was already greater than 242.
ALTER TABLE items
    ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS planning_abc_class VARCHAR(1),
    ADD COLUMN IF NOT EXISTS planning_minimum_lot BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS planning_multiple_lot BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS planning_safety_stock BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS planning_critical BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS planning_exclusive BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS planning_active BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS supplies_purchase_uom VARCHAR(20),
    ADD COLUMN IF NOT EXISTS supplies_warehouse_code BIGINT,
    ADD COLUMN IF NOT EXISTS supplies_receiving_checklist BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS supplies_harvest BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS commercial_warranty_days INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS accounting_active BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS accounting_calculate_pis_cofins BOOLEAN NOT NULL DEFAULT FALSE;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_items_planning_abc_class' AND conrelid = 'items'::regclass) THEN
        ALTER TABLE items ADD CONSTRAINT chk_items_planning_abc_class
            CHECK (planning_abc_class IS NULL OR planning_abc_class IN ('A', 'B', 'C'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_items_planning_lots_nonnegative' AND conrelid = 'items'::regclass) THEN
        ALTER TABLE items ADD CONSTRAINT chk_items_planning_lots_nonnegative
            CHECK (planning_minimum_lot >= 0 AND planning_multiple_lot >= 0 AND planning_safety_stock >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_items_commercial_warranty_days_nonnegative' AND conrelid = 'items'::regclass) THEN
        ALTER TABLE items ADD CONSTRAINT chk_items_commercial_warranty_days_nonnegative
            CHECK (commercial_warranty_days >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_items_supplies_warehouse_code_positive' AND conrelid = 'items'::regclass) THEN
        ALTER TABLE items ADD CONSTRAINT chk_items_supplies_warehouse_code_positive
            CHECK (supplies_warehouse_code IS NULL OR supplies_warehouse_code > 0);
    END IF;
END $$;

ALTER TABLE fiscal_classifications
    ADD COLUMN IF NOT EXISTS item_code BIGINT;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_fiscal_classifications_item' AND conrelid = 'fiscal_classifications'::regclass) THEN
        ALTER TABLE fiscal_classifications ADD CONSTRAINT fk_fiscal_classifications_item
            FOREIGN KEY (item_code) REFERENCES items(code);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_fiscal_classifications_item_code
    ON fiscal_classifications(item_code);
