ALTER TABLE public.accounting_posting_params
    DROP COLUMN IF EXISTS adiantamento_clientes_account_id,
    DROP COLUMN IF EXISTS adiantamento_fornecedores_account_id;
