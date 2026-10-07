-- NF-e de entrada no nível enterprise:
--  * tipo de operação de entrada por item (o "TES"): CFOP de entrada e se a
--    linha movimenta estoque / gera financeiro;
--  * pedido × nota × recebimento: a linha da nota aponta para a linha do pedido
--    de compra, e o pedido passa a saber quanto já foi FATURADO além de recebido;
--  * estoque lançado na aprovação, com o movimento gravado na linha (o que
--    impede lançar duas vezes);
--  * retenções (PIS/COFINS/CSLL/IRRF/INSS/ISS) e IBS/CBS/IS da Reforma;
--  * contabilização automática (parâmetros por empresa e vínculo do plano
--    gerencial com a conta contábil);
--  * documentos emitidos contra o CNPJ (distribuição DF-e) e cancelamento.
BEGIN;

ALTER TABLE public.fiscal_entries
    ADD COLUMN IF NOT EXISTS entry_operation_code BIGINT,
    ADD COLUMN IF NOT EXISTS base_ibscbs        NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_ibs          NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_cbs          NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_is           NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_ret_pis      NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_ret_cofins   NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_ret_csll     NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS base_irrf          NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_irrf         NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS base_ret_prev      NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_ret_prev     NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_iss_ret      NUMERIC(15,2) NOT NULL DEFAULT 0,
    -- Situação do lançamento no estoque: NAO_APLICA, PENDENTE, CONCLUIDO.
    ADD COLUMN IF NOT EXISTS stock_status       VARCHAR(12) NOT NULL DEFAULT 'PENDENTE',
    ADD COLUMN IF NOT EXISTS cancelled_at       TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS cancelled_by       UUID,
    ADD COLUMN IF NOT EXISTS cancel_reason      TEXT;

ALTER TABLE public.fiscal_entry_items
    ADD COLUMN IF NOT EXISTS cfop_entrada             VARCHAR(4),
    ADD COLUMN IF NOT EXISTS entry_operation_code     BIGINT,
    ADD COLUMN IF NOT EXISTS movimenta_estoque        BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS gera_financeiro          BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS warehouse_id             BIGINT REFERENCES public.warehouse(id),
    ADD COLUMN IF NOT EXISTS purchase_order_code      BIGINT,
    ADD COLUMN IF NOT EXISTS purchase_order_item_code BIGINT,
    -- Quanto desta linha já tinha entrado no estoque pelo recebimento físico
    -- do pedido (não entra de novo) e o movimento que esta nota gerou.
    ADD COLUMN IF NOT EXISTS qtd_recebida_antes       NUMERIC(18,6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS stock_movement_id        BIGINT,
    -- Custo de aquisição: valor contábil menos os impostos recuperáveis.
    ADD COLUMN IF NOT EXISTS custo_aquisicao          NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cst_ibscbs               VARCHAR(3),
    ADD COLUMN IF NOT EXISTS cclass_trib              VARCHAR(6),
    ADD COLUMN IF NOT EXISTS base_ibscbs              NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS aliq_ibs_uf              NUMERIC(7,4)  NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_ibs_uf             NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS aliq_ibs_mun             NUMERIC(7,4)  NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_ibs_mun            NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_ibs                NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS aliq_cbs                 NUMERIC(7,4)  NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_cbs                NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_is                 NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS gera_credito_ibscbs      BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX IF NOT EXISTS ix_fiscal_entry_items_po_line
    ON public.fiscal_entry_items (purchase_order_item_code) WHERE purchase_order_item_code IS NOT NULL;

-- Tipo de operação de entrada como "TES": o que a linha faz além do fiscal.
ALTER TABLE public.entry_operation_types
    ADD COLUMN IF NOT EXISTS movimenta_estoque BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS gera_financeiro   BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS credita_icms      BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS credita_ipi       BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS credita_pis_cofins BOOLEAN NOT NULL DEFAULT TRUE;

-- 3-way: o pedido de compra sabe quanto já foi faturado (nota aprovada).
ALTER TABLE public.purchase_order_items
    ADD COLUMN IF NOT EXISTS invoiced_qty NUMERIC(15,4) NOT NULL DEFAULT 0;

-- Plano de contas gerencial → conta contábil.
ALTER TABLE public.plano_contas
    ADD COLUMN IF NOT EXISTS accounting_account_id BIGINT REFERENCES public.accounting_accounts(id);

-- Parâmetros da contabilização automática, por empresa.
CREATE TABLE IF NOT EXISTS public.accounting_posting_params (
    enterprise_id            BIGINT PRIMARY KEY REFERENCES public.enterprise(id),
    plan_id                  BIGINT NOT NULL REFERENCES public.accounting_plans(id),
    fornecedores_account_id  BIGINT NOT NULL REFERENCES public.accounting_accounts(id),
    icms_recuperar_account_id   BIGINT REFERENCES public.accounting_accounts(id),
    ipi_recuperar_account_id    BIGINT REFERENCES public.accounting_accounts(id),
    pis_recuperar_account_id    BIGINT REFERENCES public.accounting_accounts(id),
    cofins_recuperar_account_id BIGINT REFERENCES public.accounting_accounts(id),
    ibs_recuperar_account_id    BIGINT REFERENCES public.accounting_accounts(id),
    cbs_recuperar_account_id    BIGINT REFERENCES public.accounting_accounts(id),
    irrf_recolher_account_id    BIGINT REFERENCES public.accounting_accounts(id),
    pcc_recolher_account_id     BIGINT REFERENCES public.accounting_accounts(id),
    inss_recolher_account_id    BIGINT REFERENCES public.accounting_accounts(id),
    iss_recolher_account_id     BIGINT REFERENCES public.accounting_accounts(id),
    -- Conta usada quando o plano de contas do item não tem conta contábil.
    despesa_padrao_account_id   BIGINT REFERENCES public.accounting_accounts(id),
    -- Contabilizar automaticamente na aprovação da nota.
    contabilizar_entrada     BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by               UUID
);

-- Origem do lançamento contábil, para estornar o que a nota gerou.
ALTER TABLE public.accounting_journal_entries
    ADD COLUMN IF NOT EXISTS source_type VARCHAR(20),
    ADD COLUMN IF NOT EXISTS source_id   BIGINT;
CREATE INDEX IF NOT EXISTS ix_accounting_journal_source
    ON public.accounting_journal_entries (source_type, source_id) WHERE source_type IS NOT NULL;

-- Documentos fiscais emitidos contra o CNPJ da empresa (distribuição DF-e).
CREATE TABLE IF NOT EXISTS public.fiscal_received_documents (
    id               BIGSERIAL PRIMARY KEY,
    enterprise_id    BIGINT NOT NULL REFERENCES public.enterprise(id),
    chave_acesso     VARCHAR(44) NOT NULL,
    cnpj_emitente    VARCHAR(14),
    nome_emitente    VARCHAR(200),
    numero_nf        BIGINT,
    serie            VARCHAR(3),
    data_emissao     DATE,
    valor_total      NUMERIC(15,2) NOT NULL DEFAULT 0,
    situacao         VARCHAR(20),            -- autorizada, cancelada, denegada
    manifestacao     VARCHAR(20),            -- ciencia, confirmacao, desconhecimento, nao_realizada
    xml_completo     BOOLEAN NOT NULL DEFAULT FALSE,
    versao           BIGINT,
    fiscal_entry_id  BIGINT REFERENCES public.fiscal_entries(id) ON DELETE SET NULL,
    synced_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, chave_acesso)
);
CREATE INDEX IF NOT EXISTS ix_fiscal_received_documents_tenant
    ON public.fiscal_received_documents (enterprise_id, data_emissao DESC);

ALTER TABLE public.fiscal_configs
    ADD COLUMN IF NOT EXISTS dfe_ultima_versao BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS dfe_sincronizado_em TIMESTAMPTZ;

-- Notas já aprovadas antes desta migração não têm controle de estoque nesta
-- tela: não podem ficar "pendentes" de um lançamento que nunca foi previsto.
UPDATE public.fiscal_entries SET stock_status = 'NAO_APLICA' WHERE status <> 'PENDING' AND status <> 'CONFERRED';

COMMIT;
