ALTER TABLE planning_params DROP CONSTRAINT IF EXISTS planning_params_param_number_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_planning_params_tenant_number
    ON planning_params(enterprise_id,param_number) WHERE enterprise_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_planning_params_tenant_key
    ON planning_params(enterprise_id,param_key) WHERE enterprise_id IS NOT NULL;

INSERT INTO planning_params (param_number,param_key,value,description,updated_by,enterprise_id)
SELECT template.param_number,template.param_key,template.value,template.description,template.updated_by,enterprise.id
FROM enterprise
CROSS JOIN LATERAL (
    SELECT DISTINCT ON (param_number) param_number,param_key,value,description,updated_by
    FROM planning_params ORDER BY param_number,id
) template
WHERE NOT EXISTS (SELECT 1 FROM planning_params current
    WHERE current.enterprise_id=enterprise.id AND current.param_number=template.param_number)
ON CONFLICT DO NOTHING;
