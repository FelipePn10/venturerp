-- Volta sem empresa. As cópias de período criadas na subida continuam como
-- períodos independentes (metas e saldos já apontam para elas).
DROP INDEX IF EXISTS public.ix_sales_goal_balances_enterprise;
DROP INDEX IF EXISTS public.ix_sales_goals_enterprise;
DROP INDEX IF EXISTS public.ix_sales_goal_periods_enterprise;
ALTER TABLE public.sales_goal_group_targets DROP CONSTRAINT IF EXISTS sales_goal_group_targets_empresa_key;
ALTER TABLE public.sales_goal_group_targets
    ADD CONSTRAINT sales_goal_group_targets_period_code_commercial_group_code__key UNIQUE (period_code, commercial_group_code, goal_type);
ALTER TABLE public.sales_goal_balances      DROP COLUMN IF EXISTS enterprise_code;
ALTER TABLE public.sales_goal_group_targets DROP COLUMN IF EXISTS enterprise_code;
ALTER TABLE public.sales_goals              DROP COLUMN IF EXISTS enterprise_code;
ALTER TABLE public.sales_goal_periods       DROP COLUMN IF EXISTS enterprise_code;
