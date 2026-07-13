-- Repair legacy installations that recorded migration 093 with an older
-- stock_movements projection. These columns are part of the canonical table and
-- are required by tenant attribution and reference indexes below.
ALTER TABLE stock_movements
    ADD COLUMN IF NOT EXISTS created_by UUID,
    ADD COLUMN IF NOT EXISTS reference_type VARCHAR(30),
    ADD COLUMN IF NOT EXISTS reference_code BIGINT,
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE stock_lot_balances ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);

UPDATE stock_movements movement SET enterprise_id = ue.enterprise_id
FROM user_enterprises ue
WHERE movement.created_by = ue.user_id AND movement.enterprise_id IS NULL
  AND (SELECT COUNT(*) FROM user_enterprises x WHERE x.user_id = movement.created_by) = 1;

UPDATE stock_movements SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;
UPDATE stock_lot_balances SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;

ALTER TABLE stock_lot_balances
    DROP CONSTRAINT IF EXISTS stock_lot_balances_item_code_mask_warehouse_id_lot_key;

CREATE INDEX IF NOT EXISTS idx_stock_movements_tenant_reference
    ON stock_movements (enterprise_id, reference_type, reference_code);
CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_lot_balances_tenant
    ON stock_lot_balances (enterprise_id, item_code, mask, warehouse_id, lot)
    WHERE enterprise_id IS NOT NULL;
