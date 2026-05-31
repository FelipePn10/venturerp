-- icms_reduction_target_enum already exists from migration 000123 (values: 'BASE', 'PERCENTUAL').
-- We create only icms_operation_type_enum here.
BEGIN;

CREATE TYPE icms_operation_type_enum AS ENUM ('ENTRADA', 'SAIDA', 'AMBAS', 'CUSTOS');

CREATE TABLE icms_reduction_substitutions (
    id                              BIGSERIAL PRIMARY KEY,
    -- Filtros de vinculação
    item_id                         BIGINT,
    item_mask                       VARCHAR(50),
    ncm_code                        VARCHAR(10),
    uf                              VARCHAR(2) NOT NULL,
    operation_type                  icms_operation_type_enum NOT NULL DEFAULT 'AMBAS',
    customer_id                     BIGINT,
    establishment_id                BIGINT,
    supplier_id                     BIGINT,
    invoice_type_out_id             BIGINT,
    invoice_type_in_id              BIGINT,
    market_segment_id               BIGINT,
    is_preferential                 BOOLEAN NOT NULL DEFAULT FALSE,
    -- ICMS alíquotas
    icms_pct_contrib                NUMERIC(7,4),
    icms_pct_non_contrib            NUMERIC(7,4),
    legal_device_icms_contrib_id    BIGINT,
    legal_device_icms_non_contrib_id BIGINT,
    -- Redução de ICMS
    icms_red_pct_contrib            NUMERIC(7,4),
    icms_red_target_contrib         icms_reduction_target_enum DEFAULT 'BASE',
    icms_red_pct_non_contrib        NUMERIC(7,4),
    icms_red_target_non_contrib     icms_reduction_target_enum DEFAULT 'BASE',
    legal_device_icms_red_contrib_id    BIGINT,
    legal_device_icms_red_non_contrib_id BIGINT,
    -- Diferimento de ICMS
    icms_deferral_pct               NUMERIC(7,4),
    icms_deferral_target            icms_reduction_target_enum DEFAULT 'BASE',
    legal_device_icms_deferral_id   BIGINT,
    icms_deferral_benefit_code      VARCHAR(20),
    -- Substituição tributária ICMS
    icms_subst_pct_contrib          NUMERIC(7,4),
    icms_subst_pct_non_contrib      NUMERIC(7,4),
    icms_subst_pct_contrib_uc       NUMERIC(7,4),
    icms_subst_red_pct              NUMERIC(7,4),
    icms_internal_pct               NUMERIC(7,4),
    legal_device_icms_subst_contrib_id  BIGINT,
    legal_device_icms_subst_non_contrib_id BIGINT,
    legal_device_icms_subst_red_id  BIGINT,
    mod_bc_icms_st                  VARCHAR(2),
    icms_pct_for_st_contrib         NUMERIC(7,4),
    icms_pct_for_st_non_contrib     NUMERIC(7,4),
    -- CST / CSOSN
    cst_icms_contrib                VARCHAR(3),
    cst_icms_non_contrib            VARCHAR(3),
    csosn_icms                      VARCHAR(4),
    cst_icms_contrib_dev            VARCHAR(3),
    cst_icms_non_contrib_dev        VARCHAR(3),
    cst_sit_trib_b                  VARCHAR(3),
    -- Código de benefício fiscal
    fiscal_benefit_code_contrib     VARCHAR(20),
    fiscal_benefit_code_non_contrib VARCHAR(20),
    fiscal_benefit_code             VARCHAR(20),
    -- IPI redução
    ipi_red_pct_contrib             NUMERIC(7,4),
    ipi_red_target_contrib          icms_reduction_target_enum DEFAULT 'BASE',
    ipi_red_pct_non_contrib         NUMERIC(7,4),
    ipi_red_target_non_contrib      icms_reduction_target_enum DEFAULT 'BASE',
    legal_device_ipi_contrib_id     BIGINT,
    legal_device_ipi_non_contrib_id BIGINT,
    cst_ipi_out                     VARCHAR(3),
    cst_ipi_in                      VARCHAR(3),
    -- FCI (Ficha de Conteúdo de Importação)
    fci_icms_pct                    NUMERIC(7,4),
    fci_reduce_base                 BOOLEAN NOT NULL DEFAULT FALSE,
    fci_icms_subst_pct              NUMERIC(7,4),
    fci_cst_icms                    VARCHAR(3),
    fci_use_icms_zf                 BOOLEAN NOT NULL DEFAULT FALSE,
    fci_difal_st_contrib_uc_pct     NUMERIC(7,4),
    -- Acréscimos de ICMS (FCP, outros)
    icms_add_pct_contrib            NUMERIC(7,4),
    icms_add_type_contrib           VARCHAR(10),
    icms_add_sum_aliq_contrib       BOOLEAN NOT NULL DEFAULT FALSE,
    icms_add_pct_non_contrib        NUMERIC(7,4),
    icms_add_type_non_contrib       VARCHAR(10),
    icms_add_sum_aliq_non_contrib   BOOLEAN NOT NULL DEFAULT FALSE,
    icms_st_add_pct_contrib         NUMERIC(7,4),
    icms_st_add_type_contrib        VARCHAR(10),
    icms_st_add_pct_non_contrib     NUMERIC(7,4),
    icms_st_add_type_non_contrib    VARCHAR(10),
    fcp_partition_pct               NUMERIC(7,4),
    -- DIFAL EC 87/2015
    difal_icms_red_pct              NUMERIC(7,4),
    difal_icms_type                 VARCHAR(20),
    difal_purchase_red_pct          NUMERIC(7,4),
    -- Optante Simples
    is_simples_optante              BOOLEAN NOT NULL DEFAULT FALSE,
    -- Status
    is_active                       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_icms_red_sub_uf ON icms_reduction_substitutions(uf);
CREATE INDEX idx_icms_red_sub_item ON icms_reduction_substitutions(item_id) WHERE item_id IS NOT NULL;
CREATE INDEX idx_icms_red_sub_ncm ON icms_reduction_substitutions(ncm_code) WHERE ncm_code IS NOT NULL;

COMMIT;
