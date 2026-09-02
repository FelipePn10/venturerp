DROP INDEX IF EXISTS uq_appropriation_default_tenant;
DROP INDEX IF EXISTS idx_sales_forecast_blocks_tenant;
DROP INDEX IF EXISTS uq_sales_forecasts_tenant_period;
ALTER TABLE sales_forecasts ADD CONSTRAINT sales_forecasts_item_code_mask_week_year_key UNIQUE(item_code,mask,week,year);
ALTER TABLE appropriation_tables DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE sales_forecast_blocks DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE sales_forecasts DROP COLUMN IF EXISTS enterprise_id;
