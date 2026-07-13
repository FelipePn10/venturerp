-- Repair legacy installations whose migration history contains the stock
-- migrations but whose operational tables were removed outside migrate.
CREATE TABLE IF NOT EXISTS stock_reservations (
 id BIGSERIAL PRIMARY KEY,item_code BIGINT NOT NULL,mask VARCHAR(200) NOT NULL DEFAULT '',warehouse_id BIGINT NOT NULL,
 quantity NUMERIC(15,4) NOT NULL,reference_type VARCHAR(30) NOT NULL,reference_code BIGINT NOT NULL,reference_item_code BIGINT,
 reservation_date DATE NOT NULL DEFAULT CURRENT_DATE,expiration_date DATE,status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',notes TEXT,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),created_by UUID NOT NULL
);
CREATE TABLE IF NOT EXISTS item_consumption_averages (
 id BIGSERIAL PRIMARY KEY,item_code BIGINT NOT NULL UNIQUE,avg_monthly_consumption NUMERIC(15,4) NOT NULL DEFAULT 0,
 total_consumed NUMERIC(15,4) NOT NULL DEFAULT 0,window_months INT NOT NULL DEFAULT 6,calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS stock_lots (
 id BIGSERIAL PRIMARY KEY,item_code BIGINT NOT NULL,lot VARCHAR(50) NOT NULL,heat_number VARCHAR(50),certificate VARCHAR(120),
 supplier_code BIGINT,received_at DATE,notes TEXT,created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),created_by UUID NOT NULL,UNIQUE(item_code,lot)
);
CREATE TABLE IF NOT EXISTS physical_inventories (
 id BIGSERIAL PRIMARY KEY,code BIGINT NOT NULL,description VARCHAR(200) NOT NULL,warehouse_id BIGINT NOT NULL,start_date DATE NOT NULL,
 end_date DATE,status VARCHAR(20) NOT NULL DEFAULT 'OPEN',total_items INT NOT NULL DEFAULT 0,counted_items INT NOT NULL DEFAULT 0,
 notes TEXT,created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),created_by UUID NOT NULL
);

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
