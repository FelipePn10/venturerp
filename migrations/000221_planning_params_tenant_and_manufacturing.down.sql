DROP INDEX IF EXISTS uq_planning_params_tenant_key;
DROP INDEX IF EXISTS uq_planning_params_tenant_number;
-- A unicidade global só pode ser restaurada em instalações com uma empresa.
DO $$ BEGIN
  IF (SELECT COUNT(DISTINCT enterprise_id) FROM planning_params) <= 1 THEN
    ALTER TABLE planning_params ADD CONSTRAINT planning_params_param_number_key UNIQUE(param_number);
  END IF;
END $$;
