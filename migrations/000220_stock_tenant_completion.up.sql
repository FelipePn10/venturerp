ALTER TABLE stock_reservations ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE item_consumption_averages ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE stock_lots ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE physical_inventories ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);

UPDATE stock_reservations SET enterprise_id=(SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE item_consumption_averages SET enterprise_id=(SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE stock_lots SET enterprise_id=(SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE physical_inventories SET enterprise_id=(SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;

ALTER TABLE stock_reservations ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE item_consumption_averages ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE stock_lots ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE physical_inventories ALTER COLUMN enterprise_id SET NOT NULL;

ALTER TABLE item_consumption_averages DROP CONSTRAINT IF EXISTS item_consumption_averages_item_code_key;
ALTER TABLE stock_lots DROP CONSTRAINT IF EXISTS stock_lots_item_code_lot_key;
CREATE UNIQUE INDEX uq_consumption_average_tenant ON item_consumption_averages(enterprise_id,item_code);
CREATE UNIQUE INDEX uq_stock_lots_tenant ON stock_lots(enterprise_id,item_code,lot);
CREATE INDEX idx_stock_reservations_tenant ON stock_reservations(enterprise_id,item_code,status);
CREATE INDEX idx_physical_inventories_tenant ON physical_inventories(enterprise_id,status);
