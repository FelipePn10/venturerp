ALTER TABLE sales_forecasts ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE sales_forecast_blocks ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE appropriation_tables ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);

UPDATE sales_forecasts x SET enterprise_id=resolved.enterprise_id FROM (SELECT x2.id,MIN(ue.enterprise_id) enterprise_id FROM sales_forecasts x2 JOIN user_enterprises ue ON ue.user_id=x2.created_by GROUP BY x2.id HAVING COUNT(DISTINCT ue.enterprise_id)=1) resolved WHERE x.id=resolved.id AND x.enterprise_id IS NULL;
UPDATE sales_forecast_blocks x SET enterprise_id=resolved.enterprise_id FROM (SELECT x2.id,MIN(ue.enterprise_id) enterprise_id FROM sales_forecast_blocks x2 JOIN user_enterprises ue ON ue.user_id=x2.created_by GROUP BY x2.id HAVING COUNT(DISTINCT ue.enterprise_id)=1) resolved WHERE x.id=resolved.id AND x.enterprise_id IS NULL;
UPDATE appropriation_tables x SET enterprise_id=resolved.enterprise_id FROM (SELECT x2.id,MIN(ue.enterprise_id) enterprise_id FROM appropriation_tables x2 JOIN user_enterprises ue ON ue.user_id=x2.created_by GROUP BY x2.id HAVING COUNT(DISTINCT ue.enterprise_id)=1) resolved WHERE x.id=resolved.id AND x.enterprise_id IS NULL;

UPDATE sales_forecasts SET enterprise_id=(SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise)=1;
UPDATE sales_forecast_blocks SET enterprise_id=(SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise)=1;
UPDATE appropriation_tables SET enterprise_id=(SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise)=1;
DO $$ BEGIN IF EXISTS(SELECT 1 FROM sales_forecasts WHERE enterprise_id IS NULL) OR EXISTS(SELECT 1 FROM sales_forecast_blocks WHERE enterprise_id IS NULL) OR EXISTS(SELECT 1 FROM appropriation_tables WHERE enterprise_id IS NULL) THEN RAISE EXCEPTION 'Nao foi possivel determinar a empresa dos registros legados de previsao'; END IF; END $$;

ALTER TABLE sales_forecasts ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE sales_forecast_blocks ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE appropriation_tables ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE sales_forecasts DROP CONSTRAINT IF EXISTS sales_forecasts_item_code_mask_week_year_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_sales_forecasts_tenant_period ON sales_forecasts(enterprise_id,item_code,COALESCE(mask,''),week,year);
CREATE INDEX IF NOT EXISTS idx_sales_forecast_blocks_tenant ON sales_forecast_blocks(enterprise_id,start_date,end_date);
CREATE UNIQUE INDEX IF NOT EXISTS uq_appropriation_default_tenant ON appropriation_tables(enterprise_id) WHERE is_default;
