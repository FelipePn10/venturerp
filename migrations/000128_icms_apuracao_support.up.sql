-- ─── DAPI Transfer Reasons ────────────────────────────────────────────────────
CREATE TABLE dapi_transfer_reasons (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(10) NOT NULL UNIQUE,
    reason      VARCHAR(200) NOT NULL,
    destination VARCHAR(200),
    valid_from  DATE,
    valid_to    DATE,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── ICMS Apuração Adjustment Codes (SPED tabela 5.1.1) ──────────────────────
CREATE TABLE icms_apuracao_adjustment_codes (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(10) NOT NULL,
    uf          VARCHAR(2) NOT NULL,
    description VARCHAR(250) NOT NULL,
    valid_from  DATE,
    valid_to    DATE,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (code, uf)
);

-- ─── ICMS Adjustment Codes (SPED tabelas 5.2/5.3/5.6/5.7) ───────────────────
CREATE TYPE icms_adjustment_table_ref_enum AS ENUM ('5.2', '5.3', '5.6', '5.7');

CREATE TABLE icms_adjustment_codes (
    id          BIGSERIAL PRIMARY KEY,
    uf          VARCHAR(2) NOT NULL,
    code        VARCHAR(10) NOT NULL,
    description VARCHAR(250) NOT NULL,
    table_ref   icms_adjustment_table_ref_enum NOT NULL DEFAULT '5.2',
    valid_from  DATE,
    valid_to    DATE,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (uf, code, table_ref)
);

-- ─── ICMS Apuração Lines ──────────────────────────────────────────────────────
CREATE TYPE apuracao_line_type_enum AS ENUM ('DEBITO', 'CREDITO', 'SALDO', 'DEDUCAO', 'OUTROS');

CREATE TABLE icms_apuracao_lines (
    id                              BIGSERIAL PRIMARY KEY,
    code                            VARCHAR(10) NOT NULL UNIQUE,
    description                     VARCHAR(200) NOT NULL,
    line_type                       apuracao_line_type_enum NOT NULL DEFAULT 'OUTROS',
    accepts_entries                 BOOLEAN NOT NULL DEFAULT TRUE,
    nature                          VARCHAR(100),
    apuracao_adjustment_code_id     BIGINT REFERENCES icms_apuracao_adjustment_codes(id),
    is_active                       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── ICMS Summary Entries (Lançamentos Resumo de ICMS) ───────────────────────
CREATE TABLE icms_summary_entries (
    id                  BIGSERIAL PRIMARY KEY,
    period              VARCHAR(7) NOT NULL,    -- YYYY-MM
    uf                  VARCHAR(2) NOT NULL,
    cfop_id             BIGINT REFERENCES cfops(id),
    icms_base           NUMERIC(15,2) NOT NULL DEFAULT 0,
    icms_value          NUMERIC(15,2) NOT NULL DEFAULT 0,
    icms_base_other     NUMERIC(15,2) NOT NULL DEFAULT 0,
    icms_value_other    NUMERIC(15,2) NOT NULL DEFAULT 0,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (period, uf, cfop_id)
);

-- ─── ICMS Summary Entry Notes ─────────────────────────────────────────────────
CREATE TABLE icms_summary_entry_notes (
    id                      BIGSERIAL PRIMARY KEY,
    summary_entry_id        BIGINT NOT NULL REFERENCES icms_summary_entries(id) ON DELETE CASCADE,
    note_number             VARCHAR(20) NOT NULL,
    note_series             VARCHAR(3),
    emitter_cnpj            VARCHAR(18),
    issue_date              DATE NOT NULL,
    item_value              NUMERIC(15,2) NOT NULL DEFAULT 0,
    icms_base               NUMERIC(15,2) NOT NULL DEFAULT 0,
    icms_value              NUMERIC(15,2) NOT NULL DEFAULT 0,
    observation             TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
