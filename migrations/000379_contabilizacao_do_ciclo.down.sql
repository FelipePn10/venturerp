DROP INDEX IF EXISTS public.ix_accounting_journal_tenant_source;
ALTER TABLE public.contas_pagar DROP COLUMN IF EXISTS retencao_tipo;
ALTER TABLE public.contas_bancarias DROP COLUMN IF EXISTS accounting_account_id;
ALTER TABLE public.accounting_posting_params
    DROP COLUMN IF EXISTS contabilizar_pagamentos,
    DROP COLUMN IF EXISTS contabilizar_recebimentos,
    DROP COLUMN IF EXISTS contabilizar_saidas,
    DROP COLUMN IF EXISTS banco_padrao_account_id,
    DROP COLUMN IF EXISTS juros_pagos_account_id,
    DROP COLUMN IF EXISTS descontos_obtidos_account_id,
    DROP COLUMN IF EXISTS clientes_account_id,
    DROP COLUMN IF EXISTS juros_recebidos_account_id,
    DROP COLUMN IF EXISTS descontos_concedidos_account_id,
    DROP COLUMN IF EXISTS receita_vendas_account_id,
    DROP COLUMN IF EXISTS icms_vendas_account_id,
    DROP COLUMN IF EXISTS icms_recolher_account_id,
    DROP COLUMN IF EXISTS icms_st_recolher_account_id,
    DROP COLUMN IF EXISTS ipi_recolher_account_id,
    DROP COLUMN IF EXISTS pis_vendas_account_id,
    DROP COLUMN IF EXISTS pis_recolher_account_id,
    DROP COLUMN IF EXISTS cofins_vendas_account_id,
    DROP COLUMN IF EXISTS cofins_recolher_account_id,
    DROP COLUMN IF EXISTS cmv_account_id,
    DROP COLUMN IF EXISTS estoque_account_id;
