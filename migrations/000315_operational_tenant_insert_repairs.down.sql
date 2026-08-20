DROP INDEX IF EXISTS uq_stock_lots_tenant;
CREATE UNIQUE INDEX uq_stock_lots_tenant
    ON stock_lots(enterprise_id, item_code, lot);

ALTER TABLE stock_lots DROP COLUMN IF EXISTS mask;

ALTER TABLE stock_remnants DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE cutting_plan_consumptions DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE cutting_pattern_placements DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE cutting_patterns DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE cutting_stock_pieces DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE cutting_plan_parts DROP COLUMN IF EXISTS enterprise_id;

DROP INDEX IF EXISTS cutting_plans_enterprise_code_key;
ALTER TABLE cutting_plans DROP COLUMN IF EXISTS enterprise_id;
