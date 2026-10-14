-- Contabilidade por empresa.
--
-- Plano de contas (accounting_plans → accounting_accounts) e demonstrativos não
-- tinham empresa: toda empresa do banco via, alterava e lançava no mesmo plano.
-- Lançamentos (accounting_journal_entries.empresa_id) e parâmetros de
-- contabilização (accounting_posting_params.enterprise_id) já eram por empresa,
-- mas apontavam para o plano compartilhado.
--
-- Backfill (a mesma regra dos clientes, 000384):
--   * cada empresa que já usa o plano (parâmetros, lançamentos, plano
--     financeiro, conta bancária, vínculo referencial) recebe o seu — o original
--     fica com a menor delas; as demais recebem cópia do plano com todas as
--     contas, e as referências delas passam para as contas da cópia;
--   * plano nunca usado vai para todas as empresas;
--   * demonstrativo (sem uso registrado) vai para todas as empresas, com os itens.
--
-- O plano referencial (accounting_reference_accounts) é tabela oficial e
-- continua global.

ALTER TABLE public.accounting_plans ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.accounting_plans ADD COLUMN _origem_id BIGINT;
ALTER TABLE public.accounting_accounts ADD COLUMN _origem_id BIGINT;

CREATE TEMP TABLE _uso_plano (plan_id BIGINT NOT NULL, enterprise_id BIGINT NOT NULL) ON COMMIT DROP;
INSERT INTO _uso_plano
SELECT plan_id, enterprise_id FROM public.accounting_posting_params WHERE plan_id IS NOT NULL AND enterprise_id IS NOT NULL
UNION SELECT plan_id, empresa_id FROM public.accounting_journal_entries
UNION SELECT a.plan_id, pc.enterprise_id FROM public.plano_contas pc JOIN public.accounting_accounts a ON a.id = pc.accounting_account_id WHERE pc.enterprise_id IS NOT NULL
UNION SELECT a.plan_id, cb.enterprise_id FROM public.contas_bancarias cb JOIN public.accounting_accounts a ON a.id = cb.accounting_account_id WHERE cb.enterprise_id IS NOT NULL
UNION SELECT a.plan_id, r.empresa_id FROM public.accounting_account_refs r JOIN public.accounting_accounts a ON a.id = r.account_id;
DELETE FROM _uso_plano u WHERE NOT EXISTS (SELECT 1 FROM public.enterprise e WHERE e.id = u.enterprise_id);

CREATE TEMP TABLE _destino_plano ON COMMIT DROP AS
SELECT DISTINCT plan_id, enterprise_id FROM _uso_plano
UNION
SELECT p.id, e.id FROM public.accounting_plans p CROSS JOIN public.enterprise e
 WHERE NOT EXISTS (SELECT 1 FROM _uso_plano u WHERE u.plan_id = p.id);

UPDATE public.accounting_plans p SET enterprise_id = d.dono
  FROM (SELECT plan_id, MIN(enterprise_id) AS dono FROM _destino_plano GROUP BY plan_id) d
 WHERE d.plan_id = p.id;

-- Cópias do plano e das contas.
INSERT INTO public.accounting_plans (plan_number, description, valid_from, valid_to, status, created_at, enterprise_id, _origem_id)
SELECT p.plan_number, p.description, p.valid_from, p.valid_to, p.status, p.created_at, d.enterprise_id, p.id
  FROM _destino_plano d JOIN public.accounting_plans p ON p.id = d.plan_id
 WHERE d.enterprise_id <> p.enterprise_id;

INSERT INTO public.accounting_accounts (plan_id, parent_id, account_number, description, nature_code, reduced_code,
                                        requires_cost_center, valid_from, valid_to, is_analytic, created_at, _origem_id)
SELECT np.id, NULL, a.account_number, a.description, a.nature_code, a.reduced_code,
       a.requires_cost_center, a.valid_from, a.valid_to, a.is_analytic, a.created_at, a.id
  FROM public.accounting_accounts a
  JOIN public.accounting_plans np ON np._origem_id = a.plan_id;

UPDATE public.accounting_accounts nova SET parent_id = pai.id
  FROM public.accounting_accounts orig, public.accounting_accounts pai
 WHERE nova._origem_id IS NOT NULL AND orig.id = nova._origem_id AND orig.parent_id IS NOT NULL
   AND pai._origem_id = orig.parent_id AND pai.plan_id = nova.plan_id;

CREATE TEMP TABLE _mapa_plano ON COMMIT DROP AS
SELECT _origem_id AS antigo, enterprise_id, id AS novo FROM public.accounting_plans WHERE _origem_id IS NOT NULL;
CREATE TEMP TABLE _mapa_conta ON COMMIT DROP AS
SELECT c._origem_id AS antigo, p.enterprise_id, c.id AS novo
  FROM public.accounting_accounts c JOIN public.accounting_plans p ON p.id = c.plan_id
 WHERE c._origem_id IS NOT NULL;

-- Referências de cada empresa passam para o plano/contas dela.
UPDATE public.accounting_posting_params x SET plan_id = m.novo
  FROM _mapa_plano m WHERE m.antigo = x.plan_id AND m.enterprise_id = x.enterprise_id;
DO $$
DECLARE col TEXT;
BEGIN
    FOR col IN SELECT column_name FROM information_schema.columns
                WHERE table_schema = 'public' AND table_name = 'accounting_posting_params' AND column_name LIKE '%\_account\_id'
    LOOP
        EXECUTE format('UPDATE public.accounting_posting_params x SET %1$I = m.novo FROM _mapa_conta m
                         WHERE m.antigo = x.%1$I AND m.enterprise_id = x.enterprise_id', col);
    END LOOP;
END $$;

UPDATE public.accounting_journal_entries x SET plan_id = m.novo
  FROM _mapa_plano m WHERE m.antigo = x.plan_id AND m.enterprise_id = x.empresa_id;
UPDATE public.accounting_journal_entries x SET debit_account_id = m.novo
  FROM _mapa_conta m WHERE m.antigo = x.debit_account_id AND m.enterprise_id = x.empresa_id;
UPDATE public.accounting_journal_entries x SET credit_account_id = m.novo
  FROM _mapa_conta m WHERE m.antigo = x.credit_account_id AND m.enterprise_id = x.empresa_id;
UPDATE public.plano_contas x SET accounting_account_id = m.novo
  FROM _mapa_conta m WHERE m.antigo = x.accounting_account_id AND m.enterprise_id = x.enterprise_id;
UPDATE public.contas_bancarias x SET accounting_account_id = m.novo
  FROM _mapa_conta m WHERE m.antigo = x.accounting_account_id AND m.enterprise_id = x.enterprise_id;
UPDATE public.accounting_account_refs x SET account_id = m.novo
  FROM _mapa_conta m WHERE m.antigo = x.account_id AND m.enterprise_id = x.empresa_id;

ALTER TABLE public.accounting_accounts DROP COLUMN _origem_id;
ALTER TABLE public.accounting_plans DROP COLUMN _origem_id;
ALTER TABLE public.accounting_plans ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_accounting_plans_enterprise ON public.accounting_plans (enterprise_id, plan_number);

-- Demonstrativos: um conjunto por empresa, com os itens.
ALTER TABLE public.accounting_demonstratives ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.accounting_demonstratives ADD COLUMN _origem_id BIGINT;
UPDATE public.accounting_demonstratives SET enterprise_id = (SELECT MIN(id) FROM public.enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE public.accounting_demonstratives DROP CONSTRAINT IF EXISTS accounting_demonstratives_code_key;
INSERT INTO public.accounting_demonstratives (code, description, term_text, created_at, enterprise_id, _origem_id)
SELECT d.code, d.description, d.term_text, d.created_at, e.id, d.id
  FROM public.accounting_demonstratives d CROSS JOIN public.enterprise e
 WHERE e.id <> d.enterprise_id AND d._origem_id IS NULL;
INSERT INTO public.accounting_demonstrative_items (demonstrative_id, item_code, description, formula, indicator_group,
       show_in_report, show_bold, is_result, is_100pct, sped_ecf_digit, sped_ecf_type, created_at)
SELECT n.id, i.item_code, i.description, i.formula, i.indicator_group,
       i.show_in_report, i.show_bold, i.is_result, i.is_100pct, i.sped_ecf_digit, i.sped_ecf_type, i.created_at
  FROM public.accounting_demonstrative_items i
  JOIN public.accounting_demonstratives n ON n._origem_id = i.demonstrative_id;
ALTER TABLE public.accounting_demonstratives DROP COLUMN _origem_id;
ALTER TABLE public.accounting_demonstratives ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.accounting_demonstratives ADD CONSTRAINT accounting_demonstratives_enterprise_code_key UNIQUE (enterprise_id, code);
