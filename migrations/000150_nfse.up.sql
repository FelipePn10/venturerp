-- NFS-e (Nota Fiscal de Serviços eletrônica) — modelo ABRASF, emitida via Focus.
-- The prestador (emitente) is the company from fiscal_configs; this table holds
-- the RPS data, the tomador, the serviço and the authorization result.

CREATE TABLE IF NOT EXISTS public.nfse (
    id                          BIGSERIAL PRIMARY KEY,
    numero_rps                  BIGINT,
    serie_rps                   VARCHAR(10),
    tipo_rps                    INT           NOT NULL DEFAULT 1,
    data_emissao                DATE          NOT NULL,
    status                      VARCHAR(20)   NOT NULL DEFAULT 'RASCUNHO'
                                  CHECK (status IN ('RASCUNHO', 'AUTORIZADA', 'CANCELADA', 'REJEITADA', 'PROCESSANDO')),
    natureza_operacao           INT           NOT NULL DEFAULT 1,
    optante_simples             BOOLEAN       NOT NULL DEFAULT FALSE,
    incentivador_cultural       BOOLEAN       NOT NULL DEFAULT FALSE,

    -- Tomador (cliente do serviço)
    tomador_cnpj_cpf            VARCHAR(14),
    tomador_razao_social        VARCHAR(150),
    tomador_email               VARCHAR(120),
    tomador_logradouro          VARCHAR(150),
    tomador_numero              VARCHAR(20),
    tomador_complemento         VARCHAR(60),
    tomador_bairro              VARCHAR(80),
    tomador_codigo_municipio    VARCHAR(7),
    tomador_uf                  VARCHAR(2),
    tomador_cep                 VARCHAR(8),

    -- Serviço prestado
    item_lista_servico          VARCHAR(10)   NOT NULL,
    codigo_tributario_municipio VARCHAR(20),
    discriminacao               TEXT          NOT NULL,
    codigo_municipio            VARCHAR(7)    NOT NULL,
    valor_servicos              NUMERIC(15,2) NOT NULL,
    valor_deducoes              NUMERIC(15,2) NOT NULL DEFAULT 0,
    aliquota_iss                NUMERIC(7,4)  NOT NULL DEFAULT 0,
    iss_retido                  BOOLEAN       NOT NULL DEFAULT FALSE,
    valor_iss                   NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_liquido               NUMERIC(15,2) NOT NULL DEFAULT 0,

    -- Emissão / prefeitura
    focus_ref                   VARCHAR(60),
    numero_nfse                 VARCHAR(60),
    codigo_verificacao          VARCHAR(60),
    url                         TEXT,
    xml_path                    TEXT,

    sales_order_code            BIGINT,
    notes                       TEXT,
    is_active                   BOOLEAN       NOT NULL DEFAULT TRUE,
    created_by                  UUID,
    created_at                  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nfse_status ON public.nfse (status);
CREATE INDEX IF NOT EXISTS idx_nfse_data_emissao ON public.nfse (data_emissao);
