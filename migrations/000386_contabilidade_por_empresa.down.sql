-- Volta ao plano compartilhado: cada cópia (mesmo número, descrição e início de
-- vigência) é fundida no plano de menor id; as referências voltam para as contas
-- do original (pelo número da conta) e as cópias são apagadas.
CREATE TEMP TABLE _volta_plano ON COMMIT DROP AS
SELECT c.id AS copia,
       (SELECT MIN(o.id) FROM public.accounting_plans o
         WHERE o.plan_number = c.plan_number AND o.description = c.description
           AND o.valid_from IS NOT DISTINCT FROM c.valid_from) AS original
  FROM public.accounting_plans c;
DELETE FROM _volta_plano WHERE copia = original;

CREATE TEMP TABLE _volta_conta ON COMMIT DROP AS
SELECT c.id AS copia, o.id AS original
  FROM public.accounting_accounts c
  JOIN _volta_plano vp ON vp.copia = c.plan_id
  JOIN public.accounting_accounts o ON o.plan_id = vp.original AND o.account_number = c.account_number;

UPDATE public.accounting_posting_params x SET plan_id = v.original FROM _volta_plano v WHERE v.copia = x.plan_id;
DO $$
DECLARE col TEXT;
BEGIN
    FOR col IN SELECT column_name FROM information_schema.columns
                WHERE table_schema = 'public' AND table_name = 'accounting_posting_params' AND column_name LIKE '%\_account\_id'
    LOOP
        EXECUTE format('UPDATE public.accounting_posting_params x SET %1$I = v.original FROM _volta_conta v WHERE v.copia = x.%1$I', col);
    END LOOP;
END $$;
UPDATE public.accounting_journal_entries x SET plan_id = v.original FROM _volta_plano v WHERE v.copia = x.plan_id;
UPDATE public.accounting_journal_entries x SET debit_account_id = v.original FROM _volta_conta v WHERE v.copia = x.debit_account_id;
UPDATE public.accounting_journal_entries x SET credit_account_id = v.original FROM _volta_conta v WHERE v.copia = x.credit_account_id;
UPDATE public.plano_contas x SET accounting_account_id = v.original FROM _volta_conta v WHERE v.copia = x.accounting_account_id;
UPDATE public.contas_bancarias x SET accounting_account_id = v.original FROM _volta_conta v WHERE v.copia = x.accounting_account_id;
UPDATE public.accounting_account_refs x SET account_id = v.original FROM _volta_conta v WHERE v.copia = x.account_id;

UPDATE public.accounting_accounts SET parent_id = NULL WHERE plan_id IN (SELECT copia FROM _volta_plano);
DELETE FROM public.accounting_accounts WHERE plan_id IN (SELECT copia FROM _volta_plano);
DELETE FROM public.accounting_plans WHERE id IN (SELECT copia FROM _volta_plano);
DROP INDEX IF EXISTS public.ix_accounting_plans_enterprise;
ALTER TABLE public.accounting_plans DROP COLUMN IF EXISTS enterprise_id;

DELETE FROM public.accounting_demonstrative_items i
 USING public.accounting_demonstratives d, public.accounting_demonstratives o
 WHERE d.id = i.demonstrative_id AND o.code = d.code AND o.id < d.id;
DELETE FROM public.accounting_demonstratives d
 USING public.accounting_demonstratives o
 WHERE o.code = d.code AND o.id < d.id;
ALTER TABLE public.accounting_demonstratives DROP CONSTRAINT IF EXISTS accounting_demonstratives_enterprise_code_key;
ALTER TABLE public.accounting_demonstratives ADD CONSTRAINT accounting_demonstratives_code_key UNIQUE (code);
ALTER TABLE public.accounting_demonstratives DROP COLUMN IF EXISTS enterprise_id;
