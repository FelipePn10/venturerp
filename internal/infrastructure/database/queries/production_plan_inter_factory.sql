-- name: ReplaceProductionPlanInterFactories :many
WITH deleted AS (
    DELETE FROM production_plan_inter_factories
    WHERE production_plan_inter_factories.plan_code = sqlc.arg(plan_code)
      AND production_plan_inter_factories.enterprise_id = sqlc.arg(enterprise_id)
), input AS (
    SELECT enterprises.enterprise_code, releases.auto_release
    FROM unnest(@enterprise_codes::bigint[]) WITH ORDINALITY AS enterprises(enterprise_code, position)
    JOIN unnest(@auto_releases::boolean[]) WITH ORDINALITY AS releases(auto_release, position)
      USING (position)
), inserted AS (
    INSERT INTO production_plan_inter_factories
        (plan_code, enterprise_id, source_enterprise_id, auto_release)
    SELECT
        (SELECT code FROM production_plans WHERE code = sqlc.arg(plan_code) AND enterprise_id = sqlc.arg(enterprise_id)),
        sqlc.arg(enterprise_id),
        (SELECT id FROM enterprise WHERE code = input.enterprise_code),
        input.auto_release
    FROM input
    RETURNING source_enterprise_id, auto_release
)
SELECT e.code AS enterprise_code, e.name AS enterprise_name, inserted.auto_release
FROM inserted
JOIN enterprise e ON e.id = inserted.source_enterprise_id
ORDER BY e.code;

-- name: ListProductionPlanInterFactories :many
SELECT e.code AS enterprise_code, e.name AS enterprise_name, pif.auto_release
FROM production_plan_inter_factories pif
JOIN enterprise e ON e.id = pif.source_enterprise_id
WHERE pif.plan_code = sqlc.arg(plan_code) AND pif.enterprise_id = sqlc.arg(enterprise_id)
ORDER BY e.code;
