-- Comercial (Fase 5.1): devolução de venda.
--
-- Só existia a devolução de COMPRA. A devolução do cliente era lançada à mão
-- (entrada avulsa no estoque, ajuste no título), sem limite pelo que foi
-- vendido, sem estorno de comissão nem de imposto, e a série não voltava.
--
-- O documento de devolução nasce da NF de saída autorizada e é efetivado de
-- uma de duas formas:
--  * EMISSAO_PROPRIA: a empresa emite a NF-e de entrada (tipo 0, finalidade 4)
--    referenciando a NF de saída — cliente não contribuinte;
--  * NF_CLIENTE: o cliente contribuinte emitiu a NF de devolução dele; a
--    empresa registra a chave e efetiva.
-- A efetivação devolve o estoque pelo custo da saída, abate os títulos a
-- receber em aberto da NF original (o que sobrar vira crédito do cliente),
-- estorna a comissão e o débito de imposto e devolve as séries ao estoque.

CREATE TABLE devolucoes_venda (
    id              BIGSERIAL     PRIMARY KEY,
    enterprise_id   BIGINT        NOT NULL REFERENCES enterprise(id),
    numero          BIGINT        NOT NULL,
    fiscal_exit_id  BIGINT        NOT NULL REFERENCES fiscal_exits(id),
    customer_code   BIGINT,
    modo            VARCHAR(16)   NOT NULL CHECK (modo IN ('EMISSAO_PROPRIA', 'NF_CLIENTE')),
    situacao        VARCHAR(12)   NOT NULL DEFAULT 'RASCUNHO' CHECK (situacao IN ('RASCUNHO', 'EMITIDA', 'EFETIVADA', 'CANCELADA')),
    motivo          VARCHAR(20)   NOT NULL CHECK (motivo IN ('DEFEITO', 'AVARIA_TRANSPORTE', 'ERRO_PEDIDO', 'DESISTENCIA', 'GARANTIA', 'OUTROS')),
    observacao      TEXT,
    warehouse_id    BIGINT        NOT NULL REFERENCES warehouse(id),
    -- NF do cliente (modo NF_CLIENTE).
    nf_cliente_chave  VARCHAR(44),
    nf_cliente_numero BIGINT,
    -- NF-e de entrada própria (modo EMISSAO_PROPRIA).
    nf_propria_id   BIGINT        REFERENCES fiscal_exits(id),
    valor_total     NUMERIC(15,2) NOT NULL DEFAULT 0,
    abatido         NUMERIC(15,2) NOT NULL DEFAULT 0,
    credito_cliente NUMERIC(15,2) NOT NULL DEFAULT 0,
    adiantamento_id BIGINT,
    efetivada_em    TIMESTAMPTZ,
    cancelada_em    TIMESTAMPTZ,
    motivo_cancelamento TEXT,
    created_by      UUID          REFERENCES users(id),
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_by      UUID          REFERENCES users(id),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, numero),
    UNIQUE (enterprise_id, id),
    CHECK (modo <> 'NF_CLIENTE' OR situacao = 'RASCUNHO' OR situacao = 'CANCELADA' OR length(nf_cliente_chave) = 44),
    CHECK (nf_cliente_chave IS NULL OR nf_cliente_chave ~ '^[0-9]{44}$')
);
CREATE INDEX idx_devolucoes_venda_nota ON devolucoes_venda(enterprise_id, fiscal_exit_id);

CREATE TABLE devolucoes_venda_itens (
    id                  BIGSERIAL     PRIMARY KEY,
    enterprise_id       BIGINT        NOT NULL,
    devolucao_id        BIGINT        NOT NULL,
    fiscal_exit_item_id BIGINT        NOT NULL REFERENCES fiscal_exit_items(id),
    item_code           BIGINT        REFERENCES items(code),
    quantidade          NUMERIC(15,6) NOT NULL CHECK (quantidade > 0),
    valor_produtos      NUMERIC(15,2) NOT NULL DEFAULT 0,
    base_icms           NUMERIC(15,2) NOT NULL DEFAULT 0,
    base_ipi            NUMERIC(15,2) NOT NULL DEFAULT 0,
    base_icms_st        NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_icms          NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_icms_st       NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_ipi           NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_pis           NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_cofins        NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_total         NUMERIC(15,2) NOT NULL DEFAULT 0,
    -- Séries devolvidas (ids de numeros_serie).
    series              BIGINT[]      NOT NULL DEFAULT '{}',
    custo_unitario      NUMERIC(15,6),
    stock_movement_id   BIGINT,
    FOREIGN KEY (enterprise_id, devolucao_id) REFERENCES devolucoes_venda(enterprise_id, id) ON DELETE CASCADE,
    UNIQUE (devolucao_id, fiscal_exit_item_id)
);

-- NF-e própria de entrada (tipo 0) e o vínculo com a devolução.
ALTER TABLE fiscal_exits
    ADD COLUMN tipo_documento     SMALLINT NOT NULL DEFAULT 1 CHECK (tipo_documento IN (0, 1)),
    ADD COLUMN devolucao_venda_id BIGINT;
ALTER TABLE fiscal_exit_items
    ADD COLUMN fiscal_exit_item_origem_id BIGINT REFERENCES fiscal_exit_items(id);

-- Os acertos financeiros da devolução de venda reusam fiscal_return_settlements
-- (a NF própria é a fiscal_exit; no modo NF do cliente não há NF própria).
ALTER TABLE fiscal_return_settlements
    ALTER COLUMN fiscal_exit_id DROP NOT NULL,
    ADD COLUMN devolucao_venda_id BIGINT,
    ADD COLUMN adiantamento_id    BIGINT;
ALTER TABLE fiscal_return_settlements
    ADD CONSTRAINT fiscal_return_settlements_origem_chk CHECK (fiscal_exit_id IS NOT NULL OR devolucao_venda_id IS NOT NULL);

-- Crédito do cliente gerado pela devolução: adiantamento RECEBER sem conta
-- bancária (não entrou dinheiro — não vai ao fluxo de caixa). Abate títulos
-- futuros do cliente pela aplicação do adiantamento.
ALTER TABLE adiantamentos ALTER COLUMN conta_bancaria_id DROP NOT NULL;

SELECT instalar_conferencia_de_empresa();
