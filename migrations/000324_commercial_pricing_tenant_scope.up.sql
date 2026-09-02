ALTER TABLE sales_tables ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
UPDATE sales_tables SET enterprise_id=(SELECT MIN(id) FROM enterprise)
 WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise)=1;
ALTER TABLE sales_tables ALTER COLUMN enterprise_id SET NOT NULL;
-- O codigo continua globalmente unico durante a compatibilidade porque
-- sales_orders.price_table_code referencia sales_tables(code).
CREATE INDEX IF NOT EXISTS idx_sales_tables_tenant_code ON sales_tables(enterprise_id,code);
CREATE INDEX IF NOT EXISTS idx_sales_tables_tenant_active ON sales_tables(enterprise_id,is_active,code);

ALTER TABLE sales_price_policies ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
UPDATE sales_price_policies p SET enterprise_id=(SELECT st.enterprise_id FROM sales_tables st WHERE st.id=p.sales_table_id) WHERE enterprise_id IS NULL;
ALTER TABLE sales_price_policies ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE sales_price_policies DROP CONSTRAINT IF EXISTS sales_price_policies_code_key;
DROP INDEX IF EXISTS uq_sales_price_policies_scope_priority_sequence_period;
CREATE UNIQUE INDEX uq_sales_price_policies_tenant_code ON sales_price_policies(enterprise_id,code);
CREATE UNIQUE INDEX uq_sales_price_policies_tenant_precedence_period ON sales_price_policies(enterprise_id,policy_scope,priority,sequence,COALESCE(validity_start,DATE '0001-01-01'),COALESCE(validity_end,DATE '9999-12-31'));

ALTER TABLE commercial_policies ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
UPDATE commercial_policies p SET enterprise_id=(SELECT st.enterprise_id FROM sales_tables st WHERE st.id=p.sales_table_id) WHERE enterprise_id IS NULL;
ALTER TABLE commercial_policies ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE commercial_policies DROP CONSTRAINT IF EXISTS commercial_policies_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_commercial_policies_tenant_code ON commercial_policies(enterprise_id,code);
CREATE INDEX IF NOT EXISTS idx_commercial_policies_tenant_resolution ON commercial_policies(enterprise_id,is_active,priority,sequence,code);

ALTER TABLE sales_table_price_history ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
UPDATE sales_table_price_history h SET enterprise_id=(SELECT st.enterprise_id FROM sales_tables st WHERE st.id=h.sales_table_id) WHERE enterprise_id IS NULL;
ALTER TABLE sales_table_price_history ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sales_table_price_history_tenant ON sales_table_price_history(enterprise_id,sales_table_code,item_code,created_at DESC);
