DROP TABLE IF EXISTS public.fiscal_return_settlements;
ALTER TABLE public.contas_receber DROP COLUMN IF EXISTS fornecedor_id;
ALTER TABLE public.fiscal_exit_items DROP COLUMN IF EXISTS fiscal_entry_item_id;
DROP INDEX IF EXISTS public.ix_fiscal_exits_devolucao;
ALTER TABLE public.fiscal_exits
    DROP COLUMN IF EXISTS supplier_code,
    DROP COLUMN IF EXISTS fiscal_entry_id,
    DROP COLUMN IF EXISTS nfe_referenciada,
    DROP COLUMN IF EXISTS finalidade;
