CREATE TABLE accounting_plans (
    id          BIGSERIAL PRIMARY KEY,
    plan_number INT          NOT NULL,
    description VARCHAR(200) NOT NULL,
    valid_from  DATE         NOT NULL,
    valid_to    DATE,
    status      VARCHAR(1)   NOT NULL DEFAULT 'I',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE accounting_accounts (
    id                   BIGSERIAL PRIMARY KEY,
    plan_id              BIGINT       NOT NULL REFERENCES accounting_plans(id),
    parent_id            BIGINT       REFERENCES accounting_accounts(id),
    account_number       VARCHAR(50)  NOT NULL,
    description          VARCHAR(200) NOT NULL,
    nature_code          VARCHAR(10)  NOT NULL,
    reduced_code         VARCHAR(20),
    requires_cost_center BOOL         NOT NULL DEFAULT FALSE,
    valid_from           DATE         NOT NULL,
    valid_to             DATE,
    is_analytic          BOOL         NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE accounting_reference_accounts (
    id               BIGSERIAL PRIMARY KEY,
    institution_code INT          NOT NULL,
    parent_ref_id    BIGINT       REFERENCES accounting_reference_accounts(id),
    account_number   VARCHAR(50)  NOT NULL,
    description      VARCHAR(200) NOT NULL,
    account_type     VARCHAR(1)   NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE accounting_account_refs (
    id             BIGSERIAL PRIMARY KEY,
    account_id     BIGINT NOT NULL REFERENCES accounting_accounts(id),
    ref_account_id BIGINT NOT NULL REFERENCES accounting_reference_accounts(id),
    empresa_id     INT    NOT NULL,
    cost_center_id BIGINT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE accounting_journal_entries (
    id                BIGSERIAL PRIMARY KEY,
    plan_id           BIGINT          NOT NULL REFERENCES accounting_plans(id),
    empresa_id        INT             NOT NULL,
    entry_date        DATE            NOT NULL,
    entry_number      VARCHAR(20)     NOT NULL,
    batch_number      VARCHAR(20)     NOT NULL DEFAULT '',
    debit_account_id  BIGINT          NOT NULL REFERENCES accounting_accounts(id),
    credit_account_id BIGINT          NOT NULL REFERENCES accounting_accounts(id),
    debit_cc_id       BIGINT,
    credit_cc_id      BIGINT,
    value             NUMERIC(15,2)   NOT NULL,
    history_code      VARCHAR(20)     NOT NULL DEFAULT '',
    description       TEXT            NOT NULL DEFAULT '',
    entry_type        VARCHAR(20)     NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE accounting_demonstratives (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(20)  NOT NULL UNIQUE,
    description VARCHAR(200) NOT NULL,
    term_text   TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE accounting_demonstrative_items (
    id               BIGSERIAL PRIMARY KEY,
    demonstrative_id BIGINT       NOT NULL REFERENCES accounting_demonstratives(id),
    item_code        INT          NOT NULL,
    description      VARCHAR(200) NOT NULL,
    formula          TEXT         NOT NULL DEFAULT '',
    indicator_group  VARCHAR(1)   NOT NULL DEFAULT '',
    show_in_report   BOOL         NOT NULL DEFAULT TRUE,
    show_bold        BOOL         NOT NULL DEFAULT FALSE,
    is_result        BOOL         NOT NULL DEFAULT FALSE,
    is_100pct        BOOL         NOT NULL DEFAULT FALSE,
    sped_ecf_digit   VARCHAR(5)   NOT NULL DEFAULT '',
    sped_ecf_type    VARCHAR(20)  NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
