-- ─── Cadastro de Redução, Substituição e Diferimento de ICMS/IPI ──────────────
--
-- This table links an (Item|NCM, UF, Operation, Customer/Supplier/Segment) tuple
-- to its ICMS and IPI rates, reductions, substitutions, deferrals, and CST codes.
-- The application resolves the correct record via the search hierarchy defined in
-- the business documentation (preferred item > item+customer+estab > item+customer
-- > classification+customer > item alone > classification alone).

CREATE TYPE tax_param_operation_enum AS ENUM (
    'AMBAS',
    'ENTRADA',
    'SAIDA',
    'CUSTOS'
);

CREATE TYPE icms_reduction_target_enum AS ENUM (
    'BASE',
    'PERCENTUAL'
);

CREATE TYPE icms_difal_type_enum AS ENUM (
    'TRIBUTADO',
    'ISENTO_OUTRAS',
    'NAO_CONSIDERA'
);

CREATE TYPE icms_acres_type_enum AS ENUM (
    'FUNDO_COMBATE_POBREZA',
    'OUTROS'
);

CREATE TABLE icms_ipi_tax_params (
    id                              BIGSERIAL PRIMARY KEY,

    -- ── Chaves de busca ─────────────────────────────────────────────────────
    -- Exactly one of (ncm_code, item_code) must be set
    ncm_code                        VARCHAR(10),        -- NCM / fiscal classification (broad)
    item_code                       BIGINT,             -- specific item override
    item_config_mask                VARCHAR(50),        -- configured item mask (when item_code set)
    uf                              CHAR(2) NOT NULL,
    operation_type                  tax_param_operation_enum NOT NULL,

    -- ── Optional FK filters ──────────────────────────────────────────────────
    customer_code                   BIGINT,             -- saída / ambas
    customer_establishment_code     BIGINT,             -- saída only
    -- supplier_code                BIGINT,             -- entrada / ambas — future: fornecedor module
    market_segment_id               BIGINT REFERENCES market_segments(id) ON DELETE SET NULL,
    invoice_type_exit_id            BIGINT REFERENCES invoice_types(id)   ON DELETE SET NULL,
    invoice_type_entry_id           BIGINT REFERENCES invoice_types(id)   ON DELETE SET NULL,
    tax_type_id                     BIGINT REFERENCES tax_types(id)       ON DELETE SET NULL,

    -- ── Flags ────────────────────────────────────────────────────────────────
    is_preferred                    BOOLEAN NOT NULL DEFAULT FALSE,
    is_simples_optante              BOOLEAN NOT NULL DEFAULT FALSE,

    -- ── Alíquotas ICMS ──────────────────────────────────────────────────────
    icms_pct_contrib                NUMERIC(5,2) NOT NULL DEFAULT 0,
    legal_device_icms_contrib_id    BIGINT REFERENCES legal_devices(id) ON DELETE SET NULL,
    icms_pct_non_contrib            NUMERIC(5,2) NOT NULL DEFAULT 0,
    legal_device_icms_non_contrib_id BIGINT REFERENCES legal_devices(id) ON DELETE SET NULL,

    -- ── Redução de ICMS ──────────────────────────────────────────────────────
    icms_red_pct_contrib            NUMERIC(5,2) NOT NULL DEFAULT 0,
    icms_red_target_contrib         icms_reduction_target_enum,
    legal_device_icms_red_contrib_id BIGINT REFERENCES legal_devices(id) ON DELETE SET NULL,
    icms_red_pct_non_contrib        NUMERIC(5,2) NOT NULL DEFAULT 0,
    icms_red_target_non_contrib     icms_reduction_target_enum,
    legal_device_icms_red_non_contrib_id BIGINT REFERENCES legal_devices(id) ON DELETE SET NULL,

    -- ── Diferimento de ICMS ──────────────────────────────────────────────────
    icms_deferral_pct               NUMERIC(5,2) NOT NULL DEFAULT 0,
    icms_deferral_target            icms_reduction_target_enum,
    legal_device_icms_deferral_id   BIGINT REFERENCES legal_devices(id) ON DELETE SET NULL,
    cod_benef_rbc                   VARCHAR(20),

    -- ── Substituição Tributária de ICMS ──────────────────────────────────────
    icms_subst_pct_contrib          NUMERIC(5,2) NOT NULL DEFAULT 0,
    legal_device_icms_subst_contrib_id BIGINT REFERENCES legal_devices(id) ON DELETE SET NULL,
    icms_subst_pct_non_contrib      NUMERIC(5,2) NOT NULL DEFAULT 0,
    legal_device_icms_subst_non_contrib_id BIGINT REFERENCES legal_devices(id) ON DELETE SET NULL,
    icms_subst_pct_contrib_uc       NUMERIC(5,2) NOT NULL DEFAULT 0,
    icms_subst_red_pct              NUMERIC(5,2) NOT NULL DEFAULT 0,
    legal_device_icms_subst_red_id  BIGINT REFERENCES legal_devices(id) ON DELETE SET NULL,

    -- ── Base de Cálculo do ICMS ST ───────────────────────────────────────────
    icms_internal_pct               NUMERIC(5,2) NOT NULL DEFAULT 0,
    bc_icms_st_modality             VARCHAR(5),         -- 0=Preço tabelado, 1=Lista negativa, 2=Lista positiva, 3=Lista neutra, 4=Margem Valor Agregado, 5=Pauta
    icms_pct_for_st_contrib         NUMERIC(5,2) NOT NULL DEFAULT 0,
    icms_pct_for_st_non_contrib     NUMERIC(5,2) NOT NULL DEFAULT 0,

    -- ── CST / CSOSN ──────────────────────────────────────────────────────────
    cst_situation_b                 VARCHAR(2),         -- tabela B da situação tributária
    csosn_icms                      VARCHAR(4),         -- Simples Nacional
    cst_icms_contrib                VARCHAR(3),         -- contribuinte
    cst_icms_non_contrib            VARCHAR(3),         -- não contribuinte
    cod_beneficio_fiscal            VARCHAR(20),
    cst_icms_contrib_dev            VARCHAR(3),         -- CST para devolução contrib.
    cst_icms_non_contrib_dev        VARCHAR(3),         -- CST para devolução não contrib.

    -- ── Redução de IPI ───────────────────────────────────────────────────────
    ipi_red_pct_contrib             NUMERIC(5,2) NOT NULL DEFAULT 0,
    ipi_red_target_contrib          icms_reduction_target_enum,
    legal_device_ipi_contrib_id     BIGINT REFERENCES legal_devices(id) ON DELETE SET NULL,
    ipi_red_pct_non_contrib         NUMERIC(5,2) NOT NULL DEFAULT 0,
    ipi_red_target_non_contrib      icms_reduction_target_enum,
    legal_device_ipi_non_contrib_id BIGINT REFERENCES legal_devices(id) ON DELETE SET NULL,
    cst_ipi_exit                    VARCHAR(2),
    cst_ipi_entry                   VARCHAR(2),

    -- ── FCI (Ficha de Conteúdo de Importação) ────────────────────────────────
    icms_pct_origins_1238           NUMERIC(5,2) NOT NULL DEFAULT 0,
    calc_base_red_fci               BOOLEAN      NOT NULL DEFAULT FALSE,
    icms_subst_pct_origins_1238     NUMERIC(5,2) NOT NULL DEFAULT 0,
    cst_icms_fci                    VARCHAR(3),
    uses_icms_zona_franca           BOOLEAN      NOT NULL DEFAULT FALSE,
    dif_aliq_st_contrib_uc          NUMERIC(5,2) NOT NULL DEFAULT 0,
    cod_benef_contrib               VARCHAR(20),
    cod_benef_non_contrib           VARCHAR(20),

    -- ── Acréscimos de ICMS e ICMS ST ─────────────────────────────────────────
    icms_acres_pct_contrib          NUMERIC(5,2) NOT NULL DEFAULT 0,
    icms_acres_type_contrib         icms_acres_type_enum,
    icms_acres_sum_contrib          BOOLEAN      NOT NULL DEFAULT FALSE,
    icms_acres_pct_non_contrib      NUMERIC(5,2) NOT NULL DEFAULT 0,
    icms_acres_type_non_contrib     icms_acres_type_enum,
    icms_acres_sum_non_contrib      BOOLEAN      NOT NULL DEFAULT FALSE,
    icms_st_acres_pct_contrib       NUMERIC(5,2) NOT NULL DEFAULT 0,
    icms_st_acres_type_contrib      icms_acres_type_enum,
    icms_st_acres_sum_contrib       BOOLEAN      NOT NULL DEFAULT FALSE,
    icms_st_acres_pct_non_contrib   NUMERIC(5,2) NOT NULL DEFAULT 0,
    icms_st_acres_type_non_contrib  icms_acres_type_enum,
    icms_st_acres_sum_non_contrib   BOOLEAN      NOT NULL DEFAULT FALSE,
    fcp_st_partilha_pct             NUMERIC(5,2) NOT NULL DEFAULT 0,

    -- ── Diferencial de Alíquota (EC/87) ──────────────────────────────────────
    icms_difal_red_pct              NUMERIC(5,2) NOT NULL DEFAULT 0,
    icms_difal_type                 icms_difal_type_enum,

    -- ── DIFAL em Compras (Uso e Consumo/Imobilizado) ──────────────────────────
    difal_purchase_red_pct          NUMERIC(5,2) NOT NULL DEFAULT 0,
    difal_purchase_red_target       icms_reduction_target_enum,

    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tax_params_uf           ON icms_ipi_tax_params (uf);
CREATE INDEX idx_tax_params_item         ON icms_ipi_tax_params (item_code) WHERE item_code IS NOT NULL;
CREATE INDEX idx_tax_params_ncm          ON icms_ipi_tax_params (ncm_code)  WHERE ncm_code IS NOT NULL;
CREATE INDEX idx_tax_params_customer     ON icms_ipi_tax_params (customer_code) WHERE customer_code IS NOT NULL;
CREATE INDEX idx_tax_params_operation    ON icms_ipi_tax_params (operation_type);
