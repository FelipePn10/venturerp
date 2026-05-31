BEGIN;

-- ─── Aba Adicionais do Resumo de ICMS ────────────────────────────────────────
CREATE TYPE arrecadacao_indicator_enum AS ENUM ('SEFAZ', 'JUSTICA_FEDERAL', 'JUSTICA_ESTADUAL', 'OUTROS');

CREATE TABLE icms_summary_entry_additionals (
    id                      BIGSERIAL PRIMARY KEY,
    summary_entry_id        BIGINT NOT NULL REFERENCES icms_summary_entries(id) ON DELETE CASCADE,
    sequence                INT NOT NULL,
    arrecadacao_indicator   arrecadacao_indicator_enum NOT NULL DEFAULT 'SEFAZ',
    processo                VARCHAR(100),
    arrecadacao             VARCHAR(50),
    description             TEXT,
    dief_table              VARCHAR(20),
    dief_code               VARCHAR(20),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (summary_entry_id, sequence)
);

-- ─── Notas vinculadas ao Resumo de ICMS (C197 / DRCST) ───────────────────────
-- Estende a tabela existente icms_summary_entry_notes com campos para C197
ALTER TABLE icms_summary_entry_notes
    ADD COLUMN IF NOT EXISTS adjustment_value  NUMERIC(15,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS aliquota          NUMERIC(7,4)  DEFAULT 0,
    ADD COLUMN IF NOT EXISTS calc_base         NUMERIC(15,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS other_value       NUMERIC(15,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS note_type         VARCHAR(10),   -- 'ENTRADA' | 'SAIDA'
    ADD COLUMN IF NOT EXISTS motivo_id         BIGINT REFERENCES dapi_transfer_reasons(id),
    ADD COLUMN IF NOT EXISTS visto_date        DATE,
    ADD COLUMN IF NOT EXISTS c190_obs_code     VARCHAR(20),
    ADD COLUMN IF NOT EXISTS obs_code_c190     BOOLEAN NOT NULL DEFAULT FALSE;

-- ─── Restituição/Ressarcimento/Complementação ICMS ST ─────────────────────────
-- Registros C180/C181/C185/C186 e 1250/1251 do SPED Fiscal
CREATE TYPE icms_st_restitution_type_enum AS ENUM (
    'RESTITUICAO',
    'RESSARCIMENTO',
    'COMPLEMENTACAO'
);

CREATE TABLE icms_st_restitutions (
    id                          BIGSERIAL PRIMARY KEY,
    empresa_id                  INT NOT NULL,
    period                      VARCHAR(7) NOT NULL,    -- YYYY-MM
    restitution_type            icms_st_restitution_type_enum NOT NULL,
    uf                          VARCHAR(2) NOT NULL,
    -- C180: identificação do documento original (NF de entrada com ST retida)
    orig_doc_model              VARCHAR(2),
    orig_doc_series             VARCHAR(3),
    orig_doc_number             VARCHAR(20),
    orig_doc_date               DATE,
    orig_emitter_cnpj           VARCHAR(18),
    orig_emitter_ie             VARCHAR(20),
    -- C181/C185: base cálculo, alíquota, valor ST original e ajuste
    item_id                     BIGINT,
    item_code                   VARCHAR(60),
    cfop                        VARCHAR(4),
    motivo_code                 VARCHAR(10),             -- tabela 5.7
    cst_icms                    VARCHAR(3),
    icms_st_base                NUMERIC(15,2) NOT NULL DEFAULT 0,
    icms_st_aliq                NUMERIC(7,4)  NOT NULL DEFAULT 0,
    icms_st_value               NUMERIC(15,2) NOT NULL DEFAULT 0,
    icms_st_base_restitution    NUMERIC(15,2) NOT NULL DEFAULT 0,
    icms_st_value_restitution   NUMERIC(15,2) NOT NULL DEFAULT 0,
    -- 1250/1251: consolidado por período
    icms_st_consolidated_base   NUMERIC(15,2) NOT NULL DEFAULT 0,
    icms_st_consolidated_value  NUMERIC(15,2) NOT NULL DEFAULT 0,
    -- H030 link
    h030_ind_estoque            VARCHAR(1),
    -- Status
    sped_block                  VARCHAR(5),  -- 'C180'|'C181'|'C185'|'C186'|'1250'|'1251'
    is_active                   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_icms_st_rest_period ON icms_st_restitutions(period, empresa_id);
CREATE INDEX idx_icms_st_rest_uf ON icms_st_restitutions(uf);

COMMIT;
