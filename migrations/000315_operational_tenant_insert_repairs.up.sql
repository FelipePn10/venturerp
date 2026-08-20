-- Bring the migration history in line with the tenant-aware operational schema.
ALTER TABLE cutting_plans
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);

UPDATE cutting_plans cp
SET enterprise_id = resolved.enterprise_id
FROM (
    SELECT cp2.id, MIN(ue.enterprise_id) AS enterprise_id
    FROM cutting_plans cp2
    JOIN user_enterprises ue ON ue.user_id = cp2.created_by
    WHERE cp2.enterprise_id IS NULL
    GROUP BY cp2.id
    HAVING COUNT(DISTINCT ue.enterprise_id) = 1
) resolved
WHERE cp.id = resolved.id;

UPDATE cutting_plans
SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL
  AND (SELECT COUNT(*) FROM enterprise) = 1;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM cutting_plans WHERE enterprise_id IS NULL) THEN
        RAISE EXCEPTION 'Nao foi possivel determinar a empresa de todos os planos de corte';
    END IF;
END $$;

ALTER TABLE cutting_plans
    ALTER COLUMN enterprise_id SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS cutting_plans_enterprise_code_key
    ON cutting_plans(enterprise_id, code);

ALTER TABLE cutting_plan_parts ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE cutting_stock_pieces ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE cutting_patterns ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE cutting_pattern_placements ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE cutting_plan_consumptions ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE stock_remnants ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);

UPDATE cutting_plan_parts child SET enterprise_id=parent.enterprise_id FROM cutting_plans parent WHERE child.plan_id=parent.id AND child.enterprise_id IS NULL;
UPDATE cutting_stock_pieces child SET enterprise_id=parent.enterprise_id FROM cutting_plans parent WHERE child.plan_id=parent.id AND child.enterprise_id IS NULL;
UPDATE cutting_patterns child SET enterprise_id=parent.enterprise_id FROM cutting_plans parent WHERE child.plan_id=parent.id AND child.enterprise_id IS NULL;
UPDATE cutting_pattern_placements child SET enterprise_id=pattern.enterprise_id FROM cutting_patterns pattern WHERE child.pattern_id=pattern.id AND child.enterprise_id IS NULL;
UPDATE cutting_plan_consumptions child SET enterprise_id=parent.enterprise_id FROM cutting_plans parent WHERE child.plan_id=parent.id AND child.enterprise_id IS NULL;
UPDATE stock_remnants child SET enterprise_id=parent.enterprise_id FROM cutting_plans parent WHERE child.origin_plan_id=parent.id AND child.enterprise_id IS NULL;
UPDATE stock_remnants child SET enterprise_id=resolved.enterprise_id FROM (SELECT sr.id,MIN(ue.enterprise_id) enterprise_id FROM stock_remnants sr JOIN user_enterprises ue ON ue.user_id=sr.created_by WHERE sr.enterprise_id IS NULL GROUP BY sr.id HAVING COUNT(DISTINCT ue.enterprise_id)=1) resolved WHERE child.id=resolved.id;
UPDATE stock_remnants
SET enterprise_id=(SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL
  AND (SELECT COUNT(*) FROM enterprise)=1;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM stock_remnants WHERE enterprise_id IS NULL) THEN
        RAISE EXCEPTION 'Nao foi possivel determinar a empresa de todas as sobras de estoque';
    END IF;
END $$;

ALTER TABLE cutting_plan_parts ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cutting_stock_pieces ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cutting_patterns ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cutting_pattern_placements ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cutting_plan_consumptions ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE stock_remnants ALTER COLUMN enterprise_id SET NOT NULL;

ALTER TABLE stock_lots
    ADD COLUMN IF NOT EXISTS mask VARCHAR(200) NOT NULL DEFAULT '';

DROP INDEX IF EXISTS uq_stock_lots_tenant;
CREATE UNIQUE INDEX uq_stock_lots_tenant
    ON stock_lots(enterprise_id, item_code, mask, lot);
