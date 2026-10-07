BEGIN;

ALTER TABLE public.fiscal_configs
    DROP COLUMN IF EXISTS dfe_ultima_versao,
    DROP COLUMN IF EXISTS dfe_sincronizado_em;

DROP TABLE IF EXISTS public.fiscal_received_documents;

DROP INDEX IF EXISTS public.ix_accounting_journal_source;
ALTER TABLE public.accounting_journal_entries
    DROP COLUMN IF EXISTS source_type,
    DROP COLUMN IF EXISTS source_id;

DROP TABLE IF EXISTS public.accounting_posting_params;

ALTER TABLE public.plano_contas DROP COLUMN IF EXISTS accounting_account_id;
ALTER TABLE public.purchase_order_items DROP COLUMN IF EXISTS invoiced_qty;

ALTER TABLE public.entry_operation_types
    DROP COLUMN IF EXISTS movimenta_estoque,
    DROP COLUMN IF EXISTS gera_financeiro,
    DROP COLUMN IF EXISTS credita_icms,
    DROP COLUMN IF EXISTS credita_ipi,
    DROP COLUMN IF EXISTS credita_pis_cofins;

DROP INDEX IF EXISTS public.ix_fiscal_entry_items_po_line;
ALTER TABLE public.fiscal_entry_items
    DROP COLUMN IF EXISTS cfop_entrada,
    DROP COLUMN IF EXISTS entry_operation_code,
    DROP COLUMN IF EXISTS movimenta_estoque,
    DROP COLUMN IF EXISTS gera_financeiro,
    DROP COLUMN IF EXISTS warehouse_id,
    DROP COLUMN IF EXISTS purchase_order_code,
    DROP COLUMN IF EXISTS purchase_order_item_code,
    DROP COLUMN IF EXISTS qtd_recebida_antes,
    DROP COLUMN IF EXISTS stock_movement_id,
    DROP COLUMN IF EXISTS custo_aquisicao,
    DROP COLUMN IF EXISTS cst_ibscbs,
    DROP COLUMN IF EXISTS cclass_trib,
    DROP COLUMN IF EXISTS base_ibscbs,
    DROP COLUMN IF EXISTS aliq_ibs_uf,
    DROP COLUMN IF EXISTS valor_ibs_uf,
    DROP COLUMN IF EXISTS aliq_ibs_mun,
    DROP COLUMN IF EXISTS valor_ibs_mun,
    DROP COLUMN IF EXISTS valor_ibs,
    DROP COLUMN IF EXISTS aliq_cbs,
    DROP COLUMN IF EXISTS valor_cbs,
    DROP COLUMN IF EXISTS valor_is,
    DROP COLUMN IF EXISTS gera_credito_ibscbs;

ALTER TABLE public.fiscal_entries
    DROP COLUMN IF EXISTS entry_operation_code,
    DROP COLUMN IF EXISTS base_ibscbs,
    DROP COLUMN IF EXISTS valor_ibs,
    DROP COLUMN IF EXISTS valor_cbs,
    DROP COLUMN IF EXISTS valor_is,
    DROP COLUMN IF EXISTS valor_ret_pis,
    DROP COLUMN IF EXISTS valor_ret_cofins,
    DROP COLUMN IF EXISTS valor_ret_csll,
    DROP COLUMN IF EXISTS base_irrf,
    DROP COLUMN IF EXISTS valor_irrf,
    DROP COLUMN IF EXISTS base_ret_prev,
    DROP COLUMN IF EXISTS valor_ret_prev,
    DROP COLUMN IF EXISTS valor_iss_ret,
    DROP COLUMN IF EXISTS stock_status,
    DROP COLUMN IF EXISTS cancelled_at,
    DROP COLUMN IF EXISTS cancelled_by,
    DROP COLUMN IF EXISTS cancel_reason;

COMMIT;
