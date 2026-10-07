-- Contabilização automática do resto do ciclo: pagamento e recebimento de
-- títulos (inclusive o recolhimento das retenções) e NF-e de saída (receita,
-- impostos sobre vendas e CMV).
ALTER TABLE public.accounting_posting_params
    ADD COLUMN IF NOT EXISTS contabilizar_pagamentos      BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS contabilizar_recebimentos    BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS contabilizar_saidas          BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS banco_padrao_account_id      BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS juros_pagos_account_id       BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS descontos_obtidos_account_id BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS clientes_account_id          BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS juros_recebidos_account_id   BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS descontos_concedidos_account_id BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS receita_vendas_account_id    BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS icms_vendas_account_id       BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS icms_recolher_account_id     BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS icms_st_recolher_account_id  BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS ipi_recolher_account_id      BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS pis_vendas_account_id        BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS pis_recolher_account_id      BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS cofins_vendas_account_id     BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS cofins_recolher_account_id   BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS cmv_account_id               BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS estoque_account_id           BIGINT REFERENCES public.accounting_accounts(id);

-- Cada conta bancária tem a sua conta contábil (Banco X c/ movimento).
ALTER TABLE public.contas_bancarias
    ADD COLUMN IF NOT EXISTS accounting_account_id BIGINT REFERENCES public.accounting_accounts(id);

-- O título de retenção sabe qual imposto é (para o recolhimento debitar a
-- conta certa de "a recolher"). Os já existentes são preenchidos pelo número
-- ("RET IRRF NF-...").
ALTER TABLE public.contas_pagar
    ADD COLUMN IF NOT EXISTS retencao_tipo VARCHAR(10);
UPDATE public.contas_pagar
   SET retencao_tipo = split_part(numero_documento, ' ', 2)
 WHERE tipo_documento = 'RETENCAO' AND retencao_tipo IS NULL AND numero_documento LIKE 'RET %';

CREATE INDEX IF NOT EXISTS ix_accounting_journal_tenant_source
    ON public.accounting_journal_entries (empresa_id, source_type, source_id) WHERE source_type IS NOT NULL;
