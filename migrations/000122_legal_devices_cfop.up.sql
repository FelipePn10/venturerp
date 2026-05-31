-- ─── Dispositivos Legais ──────────────────────────────────────────────────────

CREATE TABLE legal_devices (
    id          BIGSERIAL PRIMARY KEY,
    code        BIGINT      NOT NULL UNIQUE,
    type        VARCHAR(10) NOT NULL CHECK (type IN ('ICMS','IPI','LAUDO','PIS','COFINS')),
    description TEXT        NOT NULL,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── CFOP / Naturezas de Operação ─────────────────────────────────────────────

CREATE TYPE cfop_utilization_enum AS ENUM (
    'INDUSTRIALIZACAO_COMERCIO',
    'IMOBILIZADO',
    'USO_CONSUMO'
);

CREATE TYPE cfop_ind_operacao_enum AS ENUM (
    'NORMAL',
    'ENERGIA_ELETRICA',
    'TELECOMUNICACAO'
);

CREATE TYPE cfop_tipo_utilizacao_enum AS ENUM (
    'NORMAL',
    'VENDA_COMERCIAL_EXPORTADORA',
    'COMPRA_FIM_ESPECIFICO_EXPORTACAO',
    'EXPORTACAO'
);

CREATE TABLE cfops (
    id                  BIGSERIAL PRIMARY KEY,
    code                INTEGER     NOT NULL UNIQUE,  -- 1000-7999 per SEFAZ table
    description         VARCHAR(200) NOT NULL,
    description_full    TEXT,
    utilization         cfop_utilization_enum    NOT NULL DEFAULT 'INDUSTRIALIZACAO_COMERCIO',
    origem_clas_ipi     VARCHAR(6),                   -- COMPRA | VENDA
    ind_operacao        cfop_ind_operacao_enum   NOT NULL DEFAULT 'NORMAL',
    tipo_utilizacao     cfop_tipo_utilizacao_enum NOT NULL DEFAULT 'NORMAL',
    codigo_anexo_sn     VARCHAR(10),
    difal               BOOLEAN NOT NULL DEFAULT FALSE,
    doacao              BOOLEAN NOT NULL DEFAULT FALSE,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
