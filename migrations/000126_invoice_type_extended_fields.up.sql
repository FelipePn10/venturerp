-- Extended fields for invoice_types: legal device FKs, CFOP FK, SPED flags, etc.

CREATE TYPE impostos_nfe_enum AS ENUM (
    'ICMS', 'IPI', 'PIS', 'COFINS', 'ICMS_IPI', 'TODOS'
);

ALTER TABLE invoice_types
    -- NF description (shown on NF document)
    ADD COLUMN description_nf                   VARCHAR(200),
    -- Taxes applicable to this invoice type
    ADD COLUMN impostos_nfe                     impostos_nfe_enum,
    -- CFOP FK (main CFOP for this invoice type)
    ADD COLUMN cfop_id                          BIGINT REFERENCES cfops(id),
    -- Legal device FKs
    ADD COLUMN dispositivo_legal_ipi_id         BIGINT REFERENCES legal_devices(id),
    ADD COLUMN dispositivo_legal_icms_id        BIGINT REFERENCES legal_devices(id),
    ADD COLUMN dispositivo_legal_icms_st_id     BIGINT REFERENCES legal_devices(id),
    ADD COLUMN dispositivo_legal_pis_id         BIGINT REFERENCES legal_devices(id),
    ADD COLUMN dispositivo_legal_cofins_id      BIGINT REFERENCES legal_devices(id),
    -- Legal device hierarchy strings (complementary text, e.g., "Art. 1º, § 2º")
    ADD COLUMN hierarchy_ipi                    VARCHAR(300),
    ADD COLUMN hierarchy_icms                   VARCHAR(300),
    ADD COLUMN hierarchy_icms_st                VARCHAR(300),
    ADD COLUMN hierarchy_pis                    VARCHAR(300),
    ADD COLUMN hierarchy_cofins                 VARCHAR(300),
    -- IPI transfer sales table FK (for IPI on transfers)
    ADD COLUMN ipi_transfer_sales_table_id      BIGINT REFERENCES sales_tables(id),
    -- SPED/SINTEGRA/fiscal book flags
    ADD COLUMN lista_valor_contabil             BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN lista_registro_saida             BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN lista_icms_ipi                   BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN sintegra_sped_fiscal             BOOLEAN NOT NULL DEFAULT TRUE,
    -- Calculation/behavior flags
    ADD COLUMN calc_fomentar                    BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN excecao_fomentar                 BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN comp_ress_ret_st                 BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN calc_reducao                     BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN complemento_itens               BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN busca_tipo_nf                    BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN icms_st_ult_entrada              BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN somente_consulta_lotes           BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN calc_imp_ibpt                    BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN cred_presumido_icms              BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN ciap                             BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN vlr_agregado_base_subst          BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN contrato_facon                   BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN desc_icms_licitacoes             BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN sisdeclara                       BOOLEAN NOT NULL DEFAULT FALSE,
    -- Classification/reason codes
    ADD COLUMN cod_clas_trib                    VARCHAR(10),
    ADD COLUMN cod_clas_trib_trib_reg           VARCHAR(10),
    ADD COLUMN cod_motivo_rest_comp_icms_st     VARCHAR(10),
    ADD COLUMN cod_beneficio_fiscal             VARCHAR(20);
