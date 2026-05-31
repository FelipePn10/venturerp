BEGIN;

CREATE TYPE special_note_purpose_enum AS ENUM ('COMPLEMENTAR', 'AJUSTE');
CREATE TYPE special_note_status_enum AS ENUM ('RASCUNHO', 'EMITIDA', 'CANCELADA');

CREATE TABLE special_adjustment_notes (
    id                          BIGSERIAL PRIMARY KEY,
    empresa_id                  INT NOT NULL,
    purpose                     special_note_purpose_enum NOT NULL,
    status                      special_note_status_enum NOT NULL DEFAULT 'RASCUNHO',
    number                      VARCHAR(20),
    series                      VARCHAR(3),
    issue_date                  DATE NOT NULL,
    period                      VARCHAR(7) NOT NULL,    -- YYYY-MM (apuração)
    -- Tipo de nota e vínculo fiscal
    invoice_type_id             BIGINT,                 -- FK para invoice_types (se existir)
    cfop_id                     BIGINT REFERENCES cfops(id),
    icms_apuracao_line_id       BIGINT REFERENCES icms_apuracao_lines(id),
    adjustment_code_id          BIGINT REFERENCES icms_apuracao_adjustment_codes(id),
    adjustment_doc_code_id      BIGINT REFERENCES icms_adjustment_codes(id),
    history                     TEXT,
    -- Geração automática de lançamento resumo (para nota de ajuste)
    auto_generate_summary       BOOLEAN NOT NULL DEFAULT FALSE,
    generated_summary_entry_id  BIGINT REFERENCES icms_summary_entries(id),
    -- Totais
    total_value                 NUMERIC(15,2) NOT NULL DEFAULT 0,
    total_icms                  NUMERIC(15,2) NOT NULL DEFAULT 0,
    total_ipi                   NUMERIC(15,2) NOT NULL DEFAULT 0,
    observation                 TEXT,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE special_adjustment_note_items (
    id                          BIGSERIAL PRIMARY KEY,
    note_id                     BIGINT NOT NULL REFERENCES special_adjustment_notes(id) ON DELETE CASCADE,
    sequence                    INT NOT NULL,
    item_id                     BIGINT,           -- item genérico (parâmetro 29)
    item_code                   VARCHAR(60),
    description                 TEXT,
    quantity                    NUMERIC(15,4) NOT NULL DEFAULT 0,
    unit                        VARCHAR(6),
    unit_value                  NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_value                 NUMERIC(15,2) NOT NULL DEFAULT 0,
    icms_base                   NUMERIC(15,2) NOT NULL DEFAULT 0,
    icms_pct                    NUMERIC(7,4)  NOT NULL DEFAULT 0,
    icms_deferral_pct           NUMERIC(7,4)  NOT NULL DEFAULT 0,
    icms_value                  NUMERIC(15,2) NOT NULL DEFAULT 0,
    icms_deferred_value         NUMERIC(15,2) NOT NULL DEFAULT 0,
    ipi_base                    NUMERIC(15,2) NOT NULL DEFAULT 0,
    ipi_pct                     NUMERIC(7,4)  NOT NULL DEFAULT 0,
    ipi_value                   NUMERIC(15,2) NOT NULL DEFAULT 0,
    cst_icms                    VARCHAR(3),
    cst_ipi                     VARCHAR(3),
    cfop_id                     BIGINT REFERENCES cfops(id),
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (note_id, sequence)
);

CREATE INDEX idx_special_notes_empresa_period ON special_adjustment_notes(empresa_id, period);
CREATE INDEX idx_special_notes_status ON special_adjustment_notes(status);

COMMIT;
