-- Metas de venda por empresa.
--
-- Períodos, metas por representante, metas por grupo comercial e saldos não
-- tinham empresa: uma empresa lia, alterava e criava metas para o representante
-- da outra, e o relatório somava como "realizado" os pedidos de TODAS as
-- empresas. Tudo passa a ser da empresa (enterprise_code, convenção comercial);
-- itens da meta e clientes da meta de grupo seguem o pai.
--
-- Backfill:
--   * meta e saldo de representante: a empresa do representante (000390);
--   * meta de grupo: a empresa do representante dos clientes dela; sem
--     representante, a menor empresa; saldo de grupo idem pela meta do grupo;
--   * período: cada empresa que o usa recebe o seu (o original fica com a menor,
--     as demais ganham cópia com código novo e as metas/saldos dela passam para
--     a cópia); período sem uso vai para todas as empresas.

ALTER TABLE public.sales_goal_periods      ADD COLUMN IF NOT EXISTS enterprise_code BIGINT;
ALTER TABLE public.sales_goals             ADD COLUMN IF NOT EXISTS enterprise_code BIGINT;
ALTER TABLE public.sales_goal_group_targets ADD COLUMN IF NOT EXISTS enterprise_code BIGINT;
ALTER TABLE public.sales_goal_balances     ADD COLUMN IF NOT EXISTS enterprise_code BIGINT;

CREATE TEMP TABLE _menor ON COMMIT DROP AS SELECT code FROM public.enterprise ORDER BY id LIMIT 1;

UPDATE public.sales_goals g SET enterprise_code = r.enterprise_code
  FROM public.representatives r WHERE r.code = g.representative_code;
UPDATE public.sales_goals SET enterprise_code = (SELECT code FROM _menor) WHERE enterprise_code IS NULL;

UPDATE public.sales_goal_group_targets t SET enterprise_code = d.emp
  FROM (SELECT DISTINCT ON (gc.group_goal_id) gc.group_goal_id, r.enterprise_code AS emp
          FROM public.sales_goal_group_customers gc JOIN public.representatives r ON r.code = gc.representative_code
         ORDER BY gc.group_goal_id, r.enterprise_code) d
 WHERE d.group_goal_id = t.id;
UPDATE public.sales_goal_group_targets SET enterprise_code = (SELECT code FROM _menor) WHERE enterprise_code IS NULL;

UPDATE public.sales_goal_balances b SET enterprise_code = r.enterprise_code
  FROM public.representatives r WHERE r.code = b.representative_code;
UPDATE public.sales_goal_balances b SET enterprise_code = t.enterprise_code
  FROM public.sales_goal_group_targets t
 WHERE b.enterprise_code IS NULL AND t.period_code = b.period_code AND t.commercial_group_code = b.commercial_group_code;
UPDATE public.sales_goal_balances SET enterprise_code = (SELECT code FROM _menor) WHERE enterprise_code IS NULL;

-- Períodos.
CREATE TEMP TABLE _uso_periodo ON COMMIT DROP AS
SELECT period_code AS code, enterprise_code FROM public.sales_goals
UNION SELECT period_code, enterprise_code FROM public.sales_goal_group_targets
UNION SELECT period_code, enterprise_code FROM public.sales_goal_balances
UNION SELECT next_period_code, enterprise_code FROM public.sales_goal_balances WHERE next_period_code IS NOT NULL;
INSERT INTO _uso_periodo
SELECT p.code, e.code FROM public.sales_goal_periods p CROSS JOIN public.enterprise e
 WHERE NOT EXISTS (SELECT 1 FROM _uso_periodo u WHERE u.code = p.code);

UPDATE public.sales_goal_periods p SET enterprise_code = d.dono
  FROM (SELECT DISTINCT ON (u.code) u.code, u.enterprise_code AS dono
          FROM _uso_periodo u JOIN public.enterprise e ON e.code = u.enterprise_code
         ORDER BY u.code, e.id) d
 WHERE d.code = p.code;

CREATE TEMP TABLE _mapa_periodo ON COMMIT DROP AS
SELECT DISTINCT u.code AS antigo, u.enterprise_code, 0::BIGINT AS novo
  FROM _uso_periodo u JOIN public.sales_goal_periods p ON p.code = u.code
 WHERE u.enterprise_code <> p.enterprise_code;
UPDATE _mapa_periodo SET novo = nextval(pg_get_serial_sequence('public.sales_goal_periods', 'code'));

INSERT INTO public.sales_goal_periods (code, description, period_type, start_date, end_date, is_active, created_at, updated_at, enterprise_code)
SELECT m.novo, p.description, p.period_type, p.start_date, p.end_date, p.is_active, p.created_at, NOW(), m.enterprise_code
  FROM _mapa_periodo m JOIN public.sales_goal_periods p ON p.code = m.antigo;

UPDATE public.sales_goals x SET period_code = m.novo
  FROM _mapa_periodo m WHERE m.antigo = x.period_code AND m.enterprise_code = x.enterprise_code;
UPDATE public.sales_goal_group_targets x SET period_code = m.novo
  FROM _mapa_periodo m WHERE m.antigo = x.period_code AND m.enterprise_code = x.enterprise_code;
UPDATE public.sales_goal_balances x SET period_code = m.novo
  FROM _mapa_periodo m WHERE m.antigo = x.period_code AND m.enterprise_code = x.enterprise_code;
UPDATE public.sales_goal_balances x SET next_period_code = m.novo
  FROM _mapa_periodo m WHERE m.antigo = x.next_period_code AND m.enterprise_code = x.enterprise_code;

-- Instalação nova (nenhuma empresa): período sem dono não tem a quem pertencer.
DELETE FROM public.sales_goal_periods WHERE enterprise_code IS NULL;

ALTER TABLE public.sales_goal_periods       ALTER COLUMN enterprise_code SET NOT NULL;
ALTER TABLE public.sales_goals              ALTER COLUMN enterprise_code SET NOT NULL;
ALTER TABLE public.sales_goal_group_targets ALTER COLUMN enterprise_code SET NOT NULL;
ALTER TABLE public.sales_goal_balances      ALTER COLUMN enterprise_code SET NOT NULL;

-- Meta de grupo: única por empresa.
ALTER TABLE public.sales_goal_group_targets DROP CONSTRAINT IF EXISTS sales_goal_group_targets_period_code_commercial_group_code__key;
ALTER TABLE public.sales_goal_group_targets
    ADD CONSTRAINT sales_goal_group_targets_empresa_key UNIQUE (enterprise_code, period_code, commercial_group_code, goal_type);

CREATE INDEX IF NOT EXISTS ix_sales_goal_periods_enterprise ON public.sales_goal_periods (enterprise_code, start_date);
CREATE INDEX IF NOT EXISTS ix_sales_goals_enterprise ON public.sales_goals (enterprise_code, period_code);
CREATE INDEX IF NOT EXISTS ix_sales_goal_balances_enterprise ON public.sales_goal_balances (enterprise_code, period_code);
