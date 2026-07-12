ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS origin_type VARCHAR(20) NOT NULL DEFAULT 'MANUAL';
ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS allow_quantity_change BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS allow_date_change BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE production_orders ADD CONSTRAINT chk_production_orders_origin_type
    CHECK (origin_type IN ('MANUAL','MRP','KANBAN','COMMERCIAL','ASSISTANCE'));
UPDATE production_orders SET origin_type='MRP' WHERE planned_order_id IS NOT NULL AND origin_type='MANUAL';
ALTER TABLE items ADD COLUMN IF NOT EXISTS accepts_fractional_quantity BOOLEAN NOT NULL DEFAULT TRUE;
