BEGIN;

-- Bank Accounts
CREATE TABLE IF NOT EXISTS public.contas_bancarias (
    id                  BIGSERIAL PRIMARY KEY,
    banco               VARCHAR(10) NOT NULL,
    agencia             VARCHAR(10) NOT NULL,
    conta               VARCHAR(20) NOT NULL,
    digito              VARCHAR(2),
    descricao           VARCHAR(200) NOT NULL,
    titular             VARCHAR(200),
    saldo_inicial       NUMERIC(15,2) NOT NULL DEFAULT 0,
    chave_pix           VARCHAR(100),
    tipo_chave_pix      VARCHAR(20),
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID NOT NULL
);

-- Payment Terms (Condicoes de Pagamento)
CREATE TABLE IF NOT EXISTS public.condicoes_pagamento (
    id                  BIGSERIAL PRIMARY KEY,
    nome                VARCHAR(60) NOT NULL,
    parcelas            JSONB NOT NULL,
    ativo               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Payment Methods (Formas de Pagamento)
CREATE TABLE IF NOT EXISTS public.formas_pagamento (
    id                  BIGSERIAL PRIMARY KEY,
    codigo              VARCHAR(10) NOT NULL UNIQUE,
    descricao           VARCHAR(100) NOT NULL,
    ativo               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Plan of Accounts (Plano de Contas)
CREATE TABLE IF NOT EXISTS public.plano_contas (
    id                  BIGSERIAL PRIMARY KEY,
    codigo              VARCHAR(50) NOT NULL UNIQUE,
    descricao           VARCHAR(200) NOT NULL,
    tipo                VARCHAR(10) NOT NULL,
    natureza            VARCHAR(10) NOT NULL,
    parent_code         VARCHAR(50),
    nivel               INT NOT NULL DEFAULT 1,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cost Centers (Centros de Custo)
CREATE TABLE IF NOT EXISTS public.centros_custo (
    id                  BIGSERIAL PRIMARY KEY,
    codigo              VARCHAR(50) NOT NULL UNIQUE,
    descricao           VARCHAR(200) NOT NULL,
    tipo                VARCHAR(30) NOT NULL DEFAULT 'ADMINISTRATIVO',
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Contas a Pagar
CREATE TABLE IF NOT EXISTS public.contas_pagar (
    id                      BIGSERIAL PRIMARY KEY,
    numero_documento        VARCHAR(60) NOT NULL,
    tipo_documento          VARCHAR(30) NOT NULL DEFAULT 'OUTROS',
    fornecedor_id           BIGINT,
    fiscal_entry_id         BIGINT,
    purchase_order_id       BIGINT,

    data_lancamento         DATE NOT NULL DEFAULT CURRENT_DATE,
    data_emissao            DATE NOT NULL,
    data_vencimento         DATE NOT NULL,
    data_pagamento          DATE,

    valor_bruto             NUMERIC(15,2) NOT NULL CHECK (valor_bruto > 0),
    desconto                NUMERIC(15,2) NOT NULL DEFAULT 0,
    juros                   NUMERIC(15,2) NOT NULL DEFAULT 0,
    multa                   NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_pago              NUMERIC(15,2),

    parcela_numero          INT NOT NULL DEFAULT 1,
    parcela_total           INT NOT NULL DEFAULT 1,
    parcela_pai_id          BIGINT,

    conta_bancaria_id       BIGINT,
    forma_pagamento         VARCHAR(20),

    plano_contas_id         BIGINT,
    centro_custo_id         BIGINT,

    status_aprovacao        VARCHAR(20) NOT NULL DEFAULT 'PENDENTE',
    aprovado_por            UUID,
    data_aprovacao          TIMESTAMP,
    motivo_rejeicao         TEXT,

    status                  VARCHAR(20) NOT NULL DEFAULT 'PENDENTE',
    adiantamento_id         BIGINT,
    valor_adiantamento_abatido NUMERIC(15,2),

    comprovante_path        VARCHAR(500),
    observacao              TEXT,

    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    criado_por              UUID NOT NULL,
    baixado_por             UUID,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Contas a Receber
CREATE TABLE IF NOT EXISTS public.contas_receber (
    id                      BIGSERIAL PRIMARY KEY,
    numero_documento        VARCHAR(60),
    cliente_id              BIGINT,
    fiscal_exit_id          BIGINT,
    sales_order_id          BIGINT,

    data_lancamento         DATE NOT NULL DEFAULT CURRENT_DATE,
    data_emissao            DATE NOT NULL,
    data_vencimento         DATE NOT NULL,
    data_recebimento        DATE,

    valor_bruto             NUMERIC(15,2) NOT NULL CHECK (valor_bruto > 0),
    desconto                NUMERIC(15,2) NOT NULL DEFAULT 0,
    juros                   NUMERIC(15,2) NOT NULL DEFAULT 0,
    multa                   NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_recebido          NUMERIC(15,2),

    parcela_numero          INT NOT NULL DEFAULT 1,
    parcela_total           INT NOT NULL DEFAULT 1,
    parcela_pai_id          BIGINT,

    conta_bancaria_id       BIGINT,
    forma_pagamento         VARCHAR(20),

    nosso_numero            VARCHAR(30),
    linha_digitavel         TEXT,
    codigo_barras           VARCHAR(100),
    chave_pix_gerada        VARCHAR(200),

    plano_contas_id         BIGINT,
    centro_custo_id         BIGINT,

    status                  VARCHAR(20) NOT NULL DEFAULT 'PENDENTE',
    em_protesto             BOOLEAN NOT NULL DEFAULT FALSE,

    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    criado_por              UUID NOT NULL,
    baixado_por             UUID,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cash Flow (Fluxo de Caixa)
CREATE TABLE IF NOT EXISTS public.fluxo_caixa (
    id                          BIGSERIAL PRIMARY KEY,
    data                        DATE NOT NULL,
    tipo                        VARCHAR(15) NOT NULL,
    valor                       NUMERIC(15,2) NOT NULL,
    conta_bancaria_id           BIGINT,
    conta_bancaria_destino_id   BIGINT,
    contas_pagar_id             BIGINT,
    contas_receber_id           BIGINT,
    descricao                   VARCHAR(200),
    conciliado                  BOOLEAN NOT NULL DEFAULT FALSE,
    extrato_hash                VARCHAR(100),
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tax Assessment (Apuracao de Impostos)
CREATE TABLE IF NOT EXISTS public.tax_assessments (
    id                  BIGSERIAL PRIMARY KEY,
    imposto             VARCHAR(10) NOT NULL,
    competencia         VARCHAR(7) NOT NULL,
    debitos             NUMERIC(15,2) NOT NULL DEFAULT 0,
    creditos            NUMERIC(15,2) NOT NULL DEFAULT 0,
    saldo_devedor       NUMERIC(15,2) NOT NULL DEFAULT 0,
    saldo_credor        NUMERIC(15,2) NOT NULL DEFAULT 0,
    status              VARCHAR(20) NOT NULL DEFAULT 'APURAR',
    cp_id               BIGINT,
    data_vencimento     DATE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(imposto, competencia)
);

CREATE INDEX idx_cp_vencimento ON public.contas_pagar(data_vencimento);
CREATE INDEX idx_cp_status ON public.contas_pagar(status);
CREATE INDEX idx_cp_fornecedor ON public.contas_pagar(fornecedor_id);
CREATE INDEX idx_cr_vencimento ON public.contas_receber(data_vencimento);
CREATE INDEX idx_cr_status ON public.contas_receber(status);
CREATE INDEX idx_cr_cliente ON public.contas_receber(cliente_id);
CREATE INDEX idx_fc_data ON public.fluxo_caixa(data);
CREATE INDEX idx_fc_conta ON public.fluxo_caixa(conta_bancaria_id);
CREATE INDEX idx_tax_competencia ON public.tax_assessments(competencia);

COMMIT;
