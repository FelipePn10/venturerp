BEGIN;

-- CT-e (Conhecimento de Transporte Eletrônico)
CREATE TABLE IF NOT EXISTS public.fiscal_cte (
    id                      BIGSERIAL PRIMARY KEY,
    chave_acesso            VARCHAR(44),
    numero_cte              BIGINT NOT NULL,
    serie                   VARCHAR(3) NOT NULL DEFAULT '1',
    data_emissao            DATE NOT NULL,
    data_entrada            DATE NOT NULL DEFAULT CURRENT_DATE,
    cnpj_emitente           VARCHAR(14) NOT NULL,
    razao_social_emitente   VARCHAR(200) NOT NULL,
    ie_emitente             VARCHAR(14),
    uf_emitente             VARCHAR(2),
    cfop                    VARCHAR(4) NOT NULL DEFAULT '1352',
    valor_frete             NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_seguro            NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_outros            NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_total             NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_icms              NUMERIC(15,2) NOT NULL DEFAULT 0,
    base_icms               NUMERIC(15,2) NOT NULL DEFAULT 0,
    aliq_icms               NUMERIC(7,4) NOT NULL DEFAULT 0,
    cst_icms                VARCHAR(3),
    tipo_rateio             VARCHAR(10) NOT NULL DEFAULT 'VALOR',
    fiscal_entry_id         BIGINT REFERENCES public.fiscal_entries(id) ON DELETE SET NULL,
    status                  VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    xml_path                VARCHAR(500),
    notes                   TEXT,
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL
);

-- NF-e + CT-e association
CREATE TABLE IF NOT EXISTS public.cte_nfe_association (
    cte_id          BIGINT NOT NULL REFERENCES public.fiscal_cte(id) ON DELETE CASCADE,
    fiscal_entry_id BIGINT NOT NULL REFERENCES public.fiscal_entries(id) ON DELETE CASCADE,
    rateio_valor    NUMERIC(15,2) NOT NULL DEFAULT 0,
    rateio_pct      NUMERIC(7,4) NOT NULL DEFAULT 0,
    PRIMARY KEY (cte_id, fiscal_entry_id)
);

-- Carta de Correção Eletrônica (CC-e)
CREATE TABLE IF NOT EXISTS public.carta_correcao (
    id              BIGSERIAL PRIMARY KEY,
    fiscal_exit_id  BIGINT NOT NULL REFERENCES public.fiscal_exits(id) ON DELETE CASCADE,
    numero_seq      INT NOT NULL DEFAULT 1,
    texto_correcao  TEXT NOT NULL,
    focus_ref       VARCHAR(50),
    status          VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    protocolo       VARCHAR(50),
    chave_evento    VARCHAR(44),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID NOT NULL
);
CREATE INDEX idx_cce_exit ON public.carta_correcao(fiscal_exit_id);

-- Extrato Bancário (staging para conciliação OFX)
CREATE TABLE IF NOT EXISTS public.extrato_bancario (
    id                  BIGSERIAL PRIMARY KEY,
    conta_bancaria_id   BIGINT NOT NULL REFERENCES public.contas_bancarias(id),
    data_transacao      DATE NOT NULL,
    valor               NUMERIC(15,2) NOT NULL,
    tipo                VARCHAR(10) NOT NULL,
    descricao           VARCHAR(300),
    fitid               VARCHAR(100),
    extrato_hash        VARCHAR(100) NOT NULL UNIQUE,
    fluxo_caixa_id      BIGINT REFERENCES public.fluxo_caixa(id) ON DELETE SET NULL,
    conciliado          BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_extrato_conta ON public.extrato_bancario(conta_bancaria_id);
CREATE INDEX idx_extrato_data ON public.extrato_bancario(data_transacao);
CREATE INDEX idx_extrato_hash ON public.extrato_bancario(extrato_hash);

-- Focus NF-e API Logs
CREATE TABLE IF NOT EXISTS public.focus_nfe_logs (
    id              BIGSERIAL PRIMARY KEY,
    fiscal_exit_id  BIGINT REFERENCES public.fiscal_exits(id) ON DELETE SET NULL,
    endpoint        VARCHAR(200) NOT NULL,
    method          VARCHAR(10) NOT NULL,
    request_body    TEXT,
    response_body   TEXT,
    status_code     INT,
    duration_ms     INT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_focus_logs_exit ON public.focus_nfe_logs(fiscal_exit_id);

-- Permissions system
CREATE TABLE IF NOT EXISTS public.perfis_usuario (
    id      BIGSERIAL PRIMARY KEY,
    nome    VARCHAR(60) NOT NULL UNIQUE,
    ativo   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.permissoes (
    id          BIGSERIAL PRIMARY KEY,
    perfil_id   BIGINT NOT NULL REFERENCES public.perfis_usuario(id) ON DELETE CASCADE,
    modulo      VARCHAR(50) NOT NULL,
    acao        VARCHAR(20) NOT NULL,
    permitido   BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE(perfil_id, modulo, acao)
);
CREATE INDEX idx_perm_perfil ON public.permissoes(perfil_id);

CREATE TABLE IF NOT EXISTS public.usuarios_perfis (
    usuario_id  UUID NOT NULL,
    perfil_id   BIGINT NOT NULL REFERENCES public.perfis_usuario(id) ON DELETE CASCADE,
    PRIMARY KEY (usuario_id, perfil_id)
);

-- Saldo Contas view
CREATE OR REPLACE VIEW public.saldo_contas AS
SELECT
    cb.id,
    cb.descricao AS nome,
    cb.banco,
    cb.saldo_inicial +
        COALESCE(SUM(CASE
            WHEN fc.tipo = 'ENTRADA' THEN fc.valor
            WHEN fc.tipo = 'TRANSFERENCIA' AND fc.conta_bancaria_destino_id = cb.id THEN fc.valor
            ELSE 0
        END), 0) -
        COALESCE(SUM(CASE
            WHEN fc.tipo = 'SAIDA' THEN fc.valor
            WHEN fc.tipo = 'TRANSFERENCIA' AND fc.conta_bancaria_id = cb.id THEN fc.valor
            ELSE 0
        END), 0) AS saldo_atual
FROM public.contas_bancarias cb
LEFT JOIN public.fluxo_caixa fc ON fc.conta_bancaria_id = cb.id OR fc.conta_bancaria_destino_id = cb.id
WHERE cb.is_active = TRUE
GROUP BY cb.id, cb.descricao, cb.banco, cb.saldo_inicial;

-- Add motivo_cancelamento to fiscal_exits
ALTER TABLE public.fiscal_exits
    ADD COLUMN IF NOT EXISTS motivo_cancelamento TEXT,
    ADD COLUMN IF NOT EXISTS data_cancelamento   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS cancelado_por        UUID,
    ADD COLUMN IF NOT EXISTS emitida_contingencia BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS condicao_pagamento_id BIGINT,
    ADD COLUMN IF NOT EXISTS tipo_pagamento        VARCHAR(20);

-- Add condicao_pagamento_id to contas_receber (link to fiscal_exits)
ALTER TABLE public.contas_receber
    ADD COLUMN IF NOT EXISTS condicao_pagamento_id BIGINT;

-- Add fornecedor_cnpj to contas_pagar for auto-generation
ALTER TABLE public.contas_pagar
    ADD COLUMN IF NOT EXISTS fornecedor_cnpj VARCHAR(14);

-- Default perfis
INSERT INTO public.perfis_usuario (nome) VALUES ('ADMIN'), ('GESTOR'), ('OPERADOR')
ON CONFLICT (nome) DO NOTHING;

-- Default permissions for ADMIN
INSERT INTO public.permissoes (perfil_id, modulo, acao, permitido)
SELECT p.id, mod.modulo, act.acao, TRUE
FROM public.perfis_usuario p
CROSS JOIN (VALUES
    ('fiscal_entrada'), ('fiscal_saida'), ('contas_pagar'), ('contas_receber'),
    ('fluxo_caixa'), ('conciliacao'), ('apuracao'), ('relatorios'), ('configuracoes')
) AS mod(modulo)
CROSS JOIN (VALUES
    ('visualizar'), ('criar'), ('editar'), ('excluir'), ('aprovar'), ('baixar'), ('exportar')
) AS act(acao)
WHERE p.nome = 'ADMIN'
ON CONFLICT (perfil_id, modulo, acao) DO NOTHING;

COMMIT;
