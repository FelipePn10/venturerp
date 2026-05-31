-- ─── Carrier / Payment Condition / Sales Table ───────────────────────────────

-- Carrier Groups
CREATE TABLE carrier_groups (
    id          BIGSERIAL PRIMARY KEY,
    code        BIGINT       NOT NULL UNIQUE,
    description VARCHAR(100) NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Carriers (financial billing entities)
CREATE TYPE carrier_billing_type_enum AS ENUM ('CARTEIRA', 'COBRANCA_ESCRITURAL', 'BOLETO');

CREATE TABLE carriers (
    id                  BIGSERIAL PRIMARY KEY,
    code                BIGINT                   NOT NULL UNIQUE,
    description         VARCHAR(150)             NOT NULL,
    billing_type        carrier_billing_type_enum NOT NULL DEFAULT 'CARTEIRA',
    uses_credit_limit   BOOLEAN                  NOT NULL DEFAULT FALSE,
    consider_available  BOOLEAN                  NOT NULL DEFAULT FALSE,
    postpone_due_date   BOOLEAN                  NOT NULL DEFAULT FALSE,
    receipt_days        SMALLINT                 NOT NULL DEFAULT 0,
    payment_days        SMALLINT                 NOT NULL DEFAULT 0,
    is_active           BOOLEAN                  NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ              NOT NULL DEFAULT NOW()
);

-- Carrier Group ↔ Carrier (many-to-many)
CREATE TABLE carrier_group_carriers (
    carrier_group_id BIGINT NOT NULL REFERENCES carrier_groups(id) ON DELETE CASCADE,
    carrier_id       BIGINT NOT NULL REFERENCES carriers(id) ON DELETE CASCADE,
    PRIMARY KEY (carrier_group_id, carrier_id)
);

-- Payment Conditions
CREATE TYPE payment_analysis_enum AS ENUM ('SEMPRE_ANALISA', 'BLOQUEIA_SEMPRE', 'LIBERA_SEM_ANALISE');
CREATE TYPE payment_parcel_start_enum AS ENUM ('EMISSAO', 'PROXIMO_MES', 'PROXIMA_QUINZENA');

CREATE TABLE payment_conditions (
    id               BIGSERIAL PRIMARY KEY,
    code             BIGINT                  NOT NULL UNIQUE,
    description      VARCHAR(150)            NOT NULL,
    carrier_id       BIGINT                  REFERENCES carriers(id),
    analysis_type    payment_analysis_enum   NOT NULL DEFAULT 'LIBERA_SEM_ANALISE',
    parcel_start     payment_parcel_start_enum NOT NULL DEFAULT 'EMISSAO',
    expenses         NUMERIC(15,4)           NOT NULL DEFAULT 0,
    average_term     SMALLINT                NOT NULL DEFAULT 0,
    is_special       BOOLEAN                 NOT NULL DEFAULT FALSE,
    is_revenue       BOOLEAN                 NOT NULL DEFAULT FALSE,
    is_at_sight      BOOLEAN                 NOT NULL DEFAULT FALSE,
    is_active        BOOLEAN                 NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

-- Payment Condition Installments
CREATE TABLE payment_condition_installments (
    id                   BIGSERIAL PRIMARY KEY,
    payment_condition_id BIGINT       NOT NULL REFERENCES payment_conditions(id) ON DELETE CASCADE,
    installment_number   SMALLINT     NOT NULL,
    due_days             SMALLINT     NOT NULL DEFAULT 0,
    description          VARCHAR(100),
    document_type        VARCHAR(50),
    movement_type        VARCHAR(50),
    carrier_id           BIGINT       REFERENCES carriers(id),
    is_active            BOOLEAN      NOT NULL DEFAULT TRUE,
    UNIQUE (payment_condition_id, installment_number)
);

-- Sales Tables
CREATE TYPE price_formation_enum AS ENUM (
    'INFORMADO',
    'CUSTO_MEDIO',
    'CUSTO_STANDARD_TOTAL',
    'CUSTO_STANDARD_MATERIAL',
    'INFORMADO_SEM_ICMS',
    'MAT_OPER',
    'TABELA_CUSTO',
    'TRANSFERENCIA_IPI',
    'TRANSFERENCIA_UF'
);

CREATE TABLE sales_tables (
    id                 BIGSERIAL PRIMARY KEY,
    code               BIGINT              NOT NULL UNIQUE,
    description        VARCHAR(150)        NOT NULL,
    validity_start     DATE,
    validity_end       DATE,
    tolerance_min_pct  NUMERIC(5,2)        NOT NULL DEFAULT 0,
    tolerance_max_pct  NUMERIC(5,2)        NOT NULL DEFAULT 0,
    price_formation    price_formation_enum NOT NULL DEFAULT 'INFORMADO',
    decimal_places     SMALLINT            NOT NULL DEFAULT 2,
    is_active          BOOLEAN             NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);
