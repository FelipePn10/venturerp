ALTER TABLE public.accounting_posting_params
    DROP COLUMN IF EXISTS ibs_vendas_account_id,
    DROP COLUMN IF EXISTS ibs_recolher_account_id,
    DROP COLUMN IF EXISTS cbs_vendas_account_id,
    DROP COLUMN IF EXISTS cbs_recolher_account_id;
