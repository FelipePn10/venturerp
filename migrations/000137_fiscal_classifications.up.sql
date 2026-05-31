BEGIN;

-- ─── Cadastro de Classificações Fiscais ───────────────────────────────────────
-- Classificação fiscal de mercadorias (I.I., IPI, PIS, COFINS, ICMS), com NCM/CEST,
-- CSTs por imposto, alíquotas de consumo/retenção/redução, Zona Franca, atributos de
-- exportação e descrições por idioma.
CREATE TABLE IF NOT EXISTS fiscal_classifications (
    id                          BIGSERIAL PRIMARY KEY,
    code                        BIGINT        NOT NULL UNIQUE,    -- Classificação
    description                 VARCHAR(200)  NOT NULL,
    ncm                         VARCHAR(10),                      -- Nomenclatura Comum do Mercosul
    cest                        VARCHAR(10),                      -- CEST
    -- IPI
    ipi_rate                    NUMERIC(15,4) NOT NULL DEFAULT 0, -- %IPI ou valor
    ipi_indicator               VARCHAR(10)   NOT NULL DEFAULT 'PERCENTUAL'
                                CHECK (ipi_indicator IN ('PERCENTUAL','VALOR')),
    apuracao                    VARCHAR(20),                      -- periodicidade da apuração
    cst_ipi_entrada             VARCHAR(2),
    cst_ipi_saida               VARCHAR(2),
    -- PIS
    pis_rate                    NUMERIC(15,4) NOT NULL DEFAULT 0,
    pis_indicator               VARCHAR(10)   NOT NULL DEFAULT 'PERCENTUAL'
                                CHECK (pis_indicator IN ('PERCENTUAL','VALOR')),
    cst_pis_entrada             VARCHAR(2),
    cst_pis_saida               VARCHAR(2),
    -- COFINS
    cofins_rate                 NUMERIC(15,4) NOT NULL DEFAULT 0,
    cofins_indicator            VARCHAR(10)   NOT NULL DEFAULT 'PERCENTUAL'
                                CHECK (cofins_indicator IN ('PERCENTUAL','VALOR')),
    cst_cofins_entrada          VARCHAR(2),
    cst_cofins_saida            VARCHAR(2),
    cofins_majorado_pct         NUMERIC(7,4)  NOT NULL DEFAULT 0,
    -- Substituição tributária
    pis_st_pct                  NUMERIC(7,4)  NOT NULL DEFAULT 0,
    cofins_st_pct               NUMERIC(7,4)  NOT NULL DEFAULT 0,
    -- Consumo
    pis_consumo_pct             NUMERIC(7,4)  NOT NULL DEFAULT 0,
    cst_pis_consumo_entrada     VARCHAR(2),
    cst_pis_consumo_saida       VARCHAR(2),
    cofins_consumo_pct          NUMERIC(7,4)  NOT NULL DEFAULT 0,
    cst_cofins_consumo_entrada  VARCHAR(2),
    cst_cofins_consumo_saida    VARCHAR(2),
    -- Retenção
    pis_retencao_pct            NUMERIC(7,4)  NOT NULL DEFAULT 0,
    cst_pis_retencao            VARCHAR(2),
    cofins_retencao_pct         NUMERIC(7,4)  NOT NULL DEFAULT 0,
    cst_cofins_retencao         VARCHAR(2),
    -- Redução
    pis_reducao_pct             NUMERIC(7,4)  NOT NULL DEFAULT 0,
    cst_pis_reducao             VARCHAR(2),
    cofins_reducao_pct          NUMERIC(7,4)  NOT NULL DEFAULT 0,
    cst_cofins_reducao          VARCHAR(2),
    -- Zona Franca de Manaus
    desc_pis_zf_pct             NUMERIC(7,4)  NOT NULL DEFAULT 0,
    desc_cofins_zf_pct          NUMERIC(7,4)  NOT NULL DEFAULT 0,
    -- Outros
    ex_tarifario                VARCHAR(10),                      -- enquadramento NCM (Ex Tarifário)
    un_ipi                      VARCHAR(10),                      -- UN p/ IPI
    un_tributacao               VARCHAR(10),                      -- UN de Tributação
    mod_bc_icms                 VARCHAR(2),                       -- modalidade BC ICMS
    mod_bc_icms_st              VARCHAR(2),                       -- modalidade BC ICMS ST
    cod_clas_trib               VARCHAR(10),                      -- Classificação Tributária CBS/IBS
    cod_clas_trib_trib_reg      VARCHAR(10),                      -- Tributação Regular CBS/IBS
    obs_fiscal                  TEXT,                             -- infAdFisco
    is_active                   BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at                  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by                  UUID          NOT NULL,
    updated_at                  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_fiscal_classifications_ncm ON fiscal_classifications(ncm) WHERE ncm IS NOT NULL;

-- Descrições por idioma (Botão Idiomas)
CREATE TABLE IF NOT EXISTS fiscal_classification_languages (
    id                       BIGSERIAL PRIMARY KEY,
    classification_id        BIGINT       NOT NULL REFERENCES fiscal_classifications(id) ON DELETE CASCADE,
    language                 VARCHAR(20)  NOT NULL,
    description              VARCHAR(200) NOT NULL,
    UNIQUE (classification_id, language)
);

-- Atributos de Exportação da NCM (Botão Atributos de Exportação / SISCOMEX)
CREATE TABLE IF NOT EXISTS fiscal_classification_export_attributes (
    id                       BIGSERIAL PRIMARY KEY,
    classification_id        BIGINT       NOT NULL REFERENCES fiscal_classifications(id) ON DELETE CASCADE,
    code                     VARCHAR(30)  NOT NULL,   -- código do atributo (num/alfanum)
    description              VARCHAR(200),
    domain                   VARCHAR(100),            -- domínio do atributo
    start_date               DATE,
    end_date                 DATE
);

COMMIT;
