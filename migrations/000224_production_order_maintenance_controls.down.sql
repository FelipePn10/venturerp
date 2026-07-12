ALTER TABLE items DROP COLUMN IF EXISTS accepts_fractional_quantity;
ALTER TABLE production_orders DROP CONSTRAINT IF EXISTS chk_production_orders_origin_type;
ALTER TABLE production_orders DROP COLUMN IF EXISTS allow_date_change;
ALTER TABLE production_orders DROP COLUMN IF EXISTS allow_quantity_change;
ALTER TABLE production_orders DROP COLUMN IF EXISTS origin_type;
