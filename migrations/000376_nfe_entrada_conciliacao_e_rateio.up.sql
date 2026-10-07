-- NF-e de entrada: dados completos do XML, conciliação item da nota × item do
-- cadastro, plano de contas por item e parcelas com a distribuição financeira
-- por plano de contas. No contas a pagar, o título continua sendo a DUPLICATA
-- (o boleto que o banco paga inteiro) e ganha o rateio por plano de contas —
-- o modelo de "múltiplas naturezas" dos ERPs de mercado.
--
-- NF-e de saída: a linha da nota passa a lembrar de qual linha do pedido de
-- venda veio, para o faturamento parcial saber o que ainda falta faturar.
BEGIN;

-- ---------------------------------------------------------------------------
-- Cabeçalho da nota de entrada
-- ---------------------------------------------------------------------------
ALTER TABLE public.fiscal_entries
    ADD COLUMN IF NOT EXISTS natureza_operacao          VARCHAR(60),
    ADD COLUMN IF NOT EXISTS cnpj_destinatario          VARCHAR(14),
    ADD COLUMN IF NOT EXISTS protocolo                  VARCHAR(20),
    ADD COLUMN IF NOT EXISTS valor_icms_st              NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_outras               NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS modalidade_frete           VARCHAR(1),
    ADD COLUMN IF NOT EXISTS informacoes_complementares TEXT,
    -- O XML original fica guardado: é o documento fiscal que a lei manda
    -- conservar, e é dele que se reconstrói a nota quando há dúvida.
    ADD COLUMN IF NOT EXISTS xml_content                TEXT,
    -- A nota declara que não há pagamento (tPag 90: bonificação, remessa,
    -- amostra): não exige parcela nem gera contas a pagar.
    ADD COLUMN IF NOT EXISTS sem_pagamento              BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS approved_at                TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS approved_by                UUID;

-- Busca da nota pela chave (importar o mesmo XML duas vezes é recusado).
CREATE INDEX IF NOT EXISTS ix_fiscal_entries_tenant_chave
    ON public.fiscal_entries (enterprise_id, chave_acesso)
    WHERE chave_acesso IS NOT NULL AND is_active;

-- ---------------------------------------------------------------------------
-- Itens da nota de entrada
-- ---------------------------------------------------------------------------
ALTER TABLE public.fiscal_entry_items
    ADD COLUMN IF NOT EXISTS ean                 VARCHAR(14),
    ADD COLUMN IF NOT EXISTS cest                VARCHAR(7),
    ADD COLUMN IF NOT EXISTS origem_mercadoria   VARCHAR(1),
    ADD COLUMN IF NOT EXISTS valor_frete         NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_seguro        NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_desconto      NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_outras        NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS base_icms_st        NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_icms_st       NUMERIC(15,2) NOT NULL DEFAULT 0,
    -- Custo do item na nota: produto + frete + seguro + outras − desconto + IPI
    -- + ICMS-ST. É o valor que vai para o plano de contas e que o contas a
    -- pagar rateia; a soma dos itens fecha com o total da nota.
    ADD COLUMN IF NOT EXISTS valor_contabil      NUMERIC(15,2) NOT NULL DEFAULT 0,
    -- Quantidade na unidade do NOSSO cadastro (a nota vem na unidade do
    -- fornecedor; o fator vem do vínculo produto × fornecedor).
    ADD COLUMN IF NOT EXISTS fator_conversao     NUMERIC(18,8),
    ADD COLUMN IF NOT EXISTS quantidade_estoque  NUMERIC(18,6),
    ADD COLUMN IF NOT EXISTS pedido_compra_xml   VARCHAR(15),
    ADD COLUMN IF NOT EXISTS item_pedido_xml     VARCHAR(6),
    ADD COLUMN IF NOT EXISTS plano_contas_id     BIGINT REFERENCES public.plano_contas(id),
    ADD COLUMN IF NOT EXISTS centro_custo_id     BIGINT REFERENCES public.centros_custo(id);

ALTER TABLE public.fiscal_entry_items DROP CONSTRAINT IF EXISTS chk_fiscal_item_resolution_strategy;
ALTER TABLE public.fiscal_entry_items ADD CONSTRAINT chk_fiscal_item_resolution_strategy
    CHECK (resolution_strategy IS NULL OR resolution_strategy IN
        ('CODIGO_EXATO','DESCRICAO','MANUAL','NAO_RESOLVIDO','EAN','HISTORICO'));

CREATE INDEX IF NOT EXISTS ix_fiscal_entry_items_item_code
    ON public.fiscal_entry_items (item_code) WHERE item_code IS NOT NULL;

-- ---------------------------------------------------------------------------
-- Parcelas (duplicatas) da nota de entrada e sua distribuição por plano de
-- contas. Cada parcela vira UM título do contas a pagar na aprovação.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.fiscal_entry_installments (
    id               BIGSERIAL PRIMARY KEY,
    enterprise_id    BIGINT NOT NULL REFERENCES public.enterprise(id),
    fiscal_entry_id  BIGINT NOT NULL REFERENCES public.fiscal_entries(id) ON DELETE CASCADE,
    numero           INT NOT NULL,
    documento        VARCHAR(60),
    data_vencimento  DATE NOT NULL,
    valor            NUMERIC(15,2) NOT NULL CHECK (valor > 0),
    forma_pagamento  VARCHAR(20),
    -- XML (duplicata da nota), CONDICAO (condição de pagamento), MANUAL ou PADRAO.
    origem           VARCHAR(10) NOT NULL DEFAULT 'MANUAL',
    conta_pagar_id   BIGINT REFERENCES public.contas_pagar(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (fiscal_entry_id, numero)
);
CREATE INDEX IF NOT EXISTS ix_fiscal_entry_installments_tenant
    ON public.fiscal_entry_installments (enterprise_id, fiscal_entry_id);

CREATE TABLE IF NOT EXISTS public.fiscal_entry_installment_allocations (
    id               BIGSERIAL PRIMARY KEY,
    installment_id   BIGINT NOT NULL REFERENCES public.fiscal_entry_installments(id) ON DELETE CASCADE,
    plano_contas_id  BIGINT NOT NULL REFERENCES public.plano_contas(id),
    centro_custo_id  BIGINT REFERENCES public.centros_custo(id),
    valor            NUMERIC(15,2) NOT NULL CHECK (valor >= 0)
);
CREATE INDEX IF NOT EXISTS ix_fiscal_entry_installment_allocations_inst
    ON public.fiscal_entry_installment_allocations (installment_id);

-- ---------------------------------------------------------------------------
-- Rateio do título do contas a pagar por plano de contas / centro de custo.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.contas_pagar_rateios (
    id               BIGSERIAL PRIMARY KEY,
    enterprise_id    BIGINT NOT NULL REFERENCES public.enterprise(id),
    conta_pagar_id   BIGINT NOT NULL REFERENCES public.contas_pagar(id) ON DELETE CASCADE,
    plano_contas_id  BIGINT NOT NULL REFERENCES public.plano_contas(id),
    centro_custo_id  BIGINT REFERENCES public.centros_custo(id),
    valor            NUMERIC(15,2) NOT NULL CHECK (valor >= 0),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS ix_contas_pagar_rateios_titulo
    ON public.contas_pagar_rateios (conta_pagar_id);
CREATE INDEX IF NOT EXISTS ix_contas_pagar_rateios_plano
    ON public.contas_pagar_rateios (enterprise_id, plano_contas_id);

-- ---------------------------------------------------------------------------
-- NF-e de saída gerada a partir do pedido de venda
-- ---------------------------------------------------------------------------
ALTER TABLE public.fiscal_exit_items
    ADD COLUMN IF NOT EXISTS sales_order_item_code BIGINT;
-- Preço líquido do pedido (com desconto) tem mais de 2 casas; com 2 casas a
-- conferência da SEFAZ (quantidade × valor unitário = valor do produto) falha
-- em linha de quantidade grande.
ALTER TABLE public.fiscal_exit_items ALTER COLUMN unit_price TYPE NUMERIC(18,6);
CREATE INDEX IF NOT EXISTS ix_fiscal_exit_items_sales_order_item
    ON public.fiscal_exit_items (sales_order_item_code) WHERE sales_order_item_code IS NOT NULL;

COMMIT;
