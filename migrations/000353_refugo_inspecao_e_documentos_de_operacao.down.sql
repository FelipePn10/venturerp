DROP TABLE IF EXISTS operation_documents;

ALTER TABLE route_operations DROP CONSTRAINT IF EXISTS chk_route_ops_scrap;
ALTER TABLE route_operations DROP COLUMN IF EXISTS inspection_required;
ALTER TABLE route_operations DROP COLUMN IF EXISTS scrap_pct;

ALTER TABLE production_order_operations DROP COLUMN IF EXISTS planned_qty;

ALTER TABLE operations DROP CONSTRAINT IF EXISTS chk_operations_scrap;
ALTER TABLE operations DROP COLUMN IF EXISTS scrap_pct;
