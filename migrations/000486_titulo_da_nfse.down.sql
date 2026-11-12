DROP INDEX IF EXISTS public.ix_contas_receber_nfse;
ALTER TABLE public.contas_receber DROP COLUMN IF EXISTS nfse_id;
ALTER TABLE public.nfse DROP COLUMN IF EXISTS data_vencimento, DROP COLUMN IF EXISTS customer_code;
