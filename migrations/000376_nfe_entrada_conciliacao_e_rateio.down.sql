BEGIN;

DROP INDEX IF EXISTS public.ix_fiscal_exit_items_sales_order_item;
ALTER TABLE public.fiscal_exit_items DROP COLUMN IF EXISTS sales_order_item_code;
ALTER TABLE public.fiscal_exit_items ALTER COLUMN unit_price TYPE NUMERIC(15,2);

DROP TABLE IF EXISTS public.contas_pagar_rateios;
DROP TABLE IF EXISTS public.fiscal_entry_installment_allocations;
DROP TABLE IF EXISTS public.fiscal_entry_installments;

DROP INDEX IF EXISTS public.ix_fiscal_entry_items_item_code;
UPDATE public.fiscal_entry_items SET resolution_strategy = 'MANUAL'
 WHERE resolution_strategy IN ('EAN','HISTORICO');
ALTER TABLE public.fiscal_entry_items DROP CONSTRAINT IF EXISTS chk_fiscal_item_resolution_strategy;
ALTER TABLE public.fiscal_entry_items ADD CONSTRAINT chk_fiscal_item_resolution_strategy
    CHECK (resolution_strategy IS NULL OR resolution_strategy IN
        ('CODIGO_EXATO','DESCRICAO','MANUAL','NAO_RESOLVIDO'));

ALTER TABLE public.fiscal_entry_items
    DROP COLUMN IF EXISTS ean,
    DROP COLUMN IF EXISTS cest,
    DROP COLUMN IF EXISTS origem_mercadoria,
    DROP COLUMN IF EXISTS valor_frete,
    DROP COLUMN IF EXISTS valor_seguro,
    DROP COLUMN IF EXISTS valor_desconto,
    DROP COLUMN IF EXISTS valor_outras,
    DROP COLUMN IF EXISTS base_icms_st,
    DROP COLUMN IF EXISTS valor_icms_st,
    DROP COLUMN IF EXISTS valor_contabil,
    DROP COLUMN IF EXISTS fator_conversao,
    DROP COLUMN IF EXISTS quantidade_estoque,
    DROP COLUMN IF EXISTS pedido_compra_xml,
    DROP COLUMN IF EXISTS item_pedido_xml,
    DROP COLUMN IF EXISTS plano_contas_id,
    DROP COLUMN IF EXISTS centro_custo_id;

DROP INDEX IF EXISTS public.ix_fiscal_entries_tenant_chave;
ALTER TABLE public.fiscal_entries
    DROP COLUMN IF EXISTS natureza_operacao,
    DROP COLUMN IF EXISTS cnpj_destinatario,
    DROP COLUMN IF EXISTS protocolo,
    DROP COLUMN IF EXISTS valor_icms_st,
    DROP COLUMN IF EXISTS valor_outras,
    DROP COLUMN IF EXISTS modalidade_frete,
    DROP COLUMN IF EXISTS informacoes_complementares,
    DROP COLUMN IF EXISTS xml_content,
    DROP COLUMN IF EXISTS sem_pagamento,
    DROP COLUMN IF EXISTS approved_at,
    DROP COLUMN IF EXISTS approved_by;

COMMIT;
