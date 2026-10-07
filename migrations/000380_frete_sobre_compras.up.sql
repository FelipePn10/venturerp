-- 1) CT-e (fiscal_cte) não tinha coluna de empresa: todas as empresas viam os
--    CT-e umas das outras. A empresa de cada registro existente vem da nota de
--    entrada ligada; sem ela, da empresa (única) do usuário que o criou; e, se
--    ainda assim não houver, da menor empresa cadastrada (base de empresa única).
ALTER TABLE public.fiscal_cte ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
UPDATE public.fiscal_cte c SET enterprise_id = fe.enterprise_id
  FROM public.fiscal_entries fe WHERE c.enterprise_id IS NULL AND fe.id = c.fiscal_entry_id;
UPDATE public.fiscal_cte c SET enterprise_id = ue.enterprise_id
  FROM (SELECT user_id, MIN(enterprise_id) AS enterprise_id FROM public.user_enterprises GROUP BY user_id HAVING COUNT(*) = 1) ue
 WHERE c.enterprise_id IS NULL AND ue.user_id = c.created_by;
UPDATE public.fiscal_cte SET enterprise_id = (SELECT MIN(id) FROM public.enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE public.fiscal_cte ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_fiscal_cte_tenant ON public.fiscal_cte (enterprise_id, data_emissao DESC);

-- 2) Frete sobre compras: o CT-e da transportadora que trouxe a mercadoria.
--    Cobre uma ou mais NF-e de entrada; ao ser lançado, rateia o frete entre os
--    itens, complementa o custo do estoque, gera o título da transportadora e
--    contabiliza.
CREATE TABLE IF NOT EXISTS public.fiscal_freight_documents (
    id                    BIGSERIAL PRIMARY KEY,
    enterprise_id         BIGINT NOT NULL REFERENCES public.enterprise(id),
    chave_cte             VARCHAR(44),
    numero                BIGINT NOT NULL,
    serie                 VARCHAR(3) NOT NULL DEFAULT '1',
    data_emissao          DATE NOT NULL,
    cnpj_transportadora   VARCHAR(14) NOT NULL,
    nome_transportadora   VARCHAR(200) NOT NULL DEFAULT '',
    uf_transportadora     VARCHAR(2),
    supplier_code         BIGINT,
    cfop                  VARCHAR(4),
    valor_frete           NUMERIC(15,2) NOT NULL CHECK (valor_frete > 0),
    base_icms             NUMERIC(15,2) NOT NULL DEFAULT 0,
    aliq_icms             NUMERIC(7,4)  NOT NULL DEFAULT 0,
    valor_icms            NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (valor_icms >= 0),
    credita_icms          BOOLEAN NOT NULL DEFAULT TRUE,
    tipo_rateio           VARCHAR(12) NOT NULL DEFAULT 'VALOR' CHECK (tipo_rateio IN ('VALOR','QUANTIDADE','PESO')),
    data_vencimento       DATE NOT NULL,
    status                VARCHAR(12) NOT NULL DEFAULT 'PENDENTE' CHECK (status IN ('PENDENTE','LANCADO','CANCELADO')),
    conta_pagar_id        BIGINT REFERENCES public.contas_pagar(id) ON DELETE SET NULL,
    xml_content           TEXT,
    observacao            TEXT,
    created_by            UUID NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lancado_em            TIMESTAMPTZ,
    lancado_por           UUID,
    cancelado_em          TIMESTAMPTZ,
    cancel_reason         TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_fiscal_freight_chave
    ON public.fiscal_freight_documents (enterprise_id, chave_cte) WHERE chave_cte IS NOT NULL AND status <> 'CANCELADO';
CREATE INDEX IF NOT EXISTS ix_fiscal_freight_tenant ON public.fiscal_freight_documents (enterprise_id, data_emissao DESC);

CREATE TABLE IF NOT EXISTS public.fiscal_freight_document_entries (
    freight_id      BIGINT NOT NULL REFERENCES public.fiscal_freight_documents(id) ON DELETE CASCADE,
    fiscal_entry_id BIGINT NOT NULL REFERENCES public.fiscal_entries(id),
    enterprise_id   BIGINT NOT NULL REFERENCES public.enterprise(id),
    PRIMARY KEY (freight_id, fiscal_entry_id)
);

CREATE TABLE IF NOT EXISTS public.fiscal_freight_allocations (
    id                   BIGSERIAL PRIMARY KEY,
    enterprise_id        BIGINT NOT NULL REFERENCES public.enterprise(id),
    freight_id           BIGINT NOT NULL REFERENCES public.fiscal_freight_documents(id) ON DELETE CASCADE,
    fiscal_entry_id      BIGINT NOT NULL REFERENCES public.fiscal_entries(id),
    fiscal_entry_item_id BIGINT NOT NULL REFERENCES public.fiscal_entry_items(id),
    item_code            BIGINT,
    warehouse_id         BIGINT,
    plano_contas_id      BIGINT,
    centro_custo_id      BIGINT,
    valor                NUMERIC(15,2) NOT NULL,  -- parte do custo do frete (sem o ICMS creditado)
    valor_estoque        NUMERIC(15,2) NOT NULL DEFAULT 0, -- complemento do custo do que ainda está no estoque
    valor_despesa        NUMERIC(15,2) NOT NULL DEFAULT 0, -- parte do que já foi consumido
    stock_movement_id    BIGINT
);
CREATE INDEX IF NOT EXISTS ix_fiscal_freight_alloc ON public.fiscal_freight_allocations (freight_id);

-- Título da transportadora sabe de qual frete veio (o pagamento baixa
-- Fornecedores quando o frete foi contabilizado).
ALTER TABLE public.contas_pagar ADD COLUMN IF NOT EXISTS freight_document_id BIGINT REFERENCES public.fiscal_freight_documents(id) ON DELETE SET NULL;
