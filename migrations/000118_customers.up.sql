-- ─── Customer Register ────────────────────────────────────────────────────────

CREATE TYPE document_type_enum AS ENUM ('CNPJ', 'CPF', 'ESTRANGEIRO', 'ISENTO');
CREATE TYPE customer_address_type_enum AS ENUM ('COBRANCA', 'ENTREGA', 'COMERCIAL', 'OUTRO');
CREATE TYPE payment_condition_visibility_enum AS ENUM (
    'SOMENTE_VINCULADOS',
    'VINCULADOS_E_NENHUM',
    'TODOS'
);

CREATE TABLE customers (
    id                      BIGSERIAL PRIMARY KEY,
    -- Corporate concept: a corporate is a logical group, establishments are branches
    code                    BIGINT       NOT NULL UNIQUE,
    corporate_code          BIGINT,                          -- NULL = is itself a corporate root
    is_corporate            BOOLEAN      NOT NULL DEFAULT FALSE,
    -- Basic identification
    name                    VARCHAR(200) NOT NULL,
    trade_name              VARCHAR(200),
    document_type           document_type_enum NOT NULL DEFAULT 'CNPJ',
    document_number         VARCHAR(20)  NOT NULL,
    state_registration      VARCHAR(30),
    municipal_registration  VARCHAR(30),
    suframa_code            VARCHAR(20),
    suframa_expiry          DATE,
    -- Classification
    region_id               BIGINT       REFERENCES regions(id),
    market_segment_id       BIGINT       REFERENCES market_segments(id),
    customer_type_id        BIGINT       REFERENCES customer_types(id),
    -- Commercial
    payment_condition_id    BIGINT       REFERENCES payment_conditions(id),
    sales_table_id          BIGINT       REFERENCES sales_tables(id),
    carrier_id              BIGINT       REFERENCES carriers(id),
    carrier_group_id        BIGINT       REFERENCES carrier_groups(id),
    invoice_type_id         BIGINT       REFERENCES invoice_types(id),
    tax_type_id             BIGINT       REFERENCES tax_types(id),
    payment_cond_visibility payment_condition_visibility_enum NOT NULL DEFAULT 'TODOS',
    -- Financial limits
    credit_limit            NUMERIC(15,2) NOT NULL DEFAULT 0,
    -- Contact
    website                 VARCHAR(255),
    -- Status
    is_active               BOOLEAN      NOT NULL DEFAULT TRUE,
    blocked                 BOOLEAN      NOT NULL DEFAULT FALSE,
    block_reason            TEXT,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by              UUID         NOT NULL,
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Customer Addresses
CREATE TABLE customer_addresses (
    id           BIGSERIAL PRIMARY KEY,
    customer_id  BIGINT                    NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    address_type customer_address_type_enum NOT NULL DEFAULT 'COMERCIAL',
    zip_code     VARCHAR(10),
    street       VARCHAR(200),
    number       VARCHAR(20),
    complement   VARCHAR(100),
    neighborhood VARCHAR(100),
    city         VARCHAR(100),
    uf           CHAR(2),
    country      VARCHAR(60)               NOT NULL DEFAULT 'Brasil',
    is_default   BOOLEAN                   NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ               NOT NULL DEFAULT NOW()
);

-- Customer Contacts
CREATE TABLE customer_contacts (
    id               BIGSERIAL PRIMARY KEY,
    customer_id      BIGINT       NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    contact_type_id  BIGINT       REFERENCES customer_contact_types(id),
    name             VARCHAR(150) NOT NULL,
    email            VARCHAR(200),
    phone            VARCHAR(30),
    mobile           VARCHAR(30),
    position         VARCHAR(100),
    is_primary       BOOLEAN      NOT NULL DEFAULT FALSE,
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Ensure only one default address per customer per type
CREATE UNIQUE INDEX ux_customer_default_address
    ON customer_addresses(customer_id, address_type)
    WHERE is_default = TRUE;

-- Index for corporate lookups
CREATE INDEX ix_customers_corporate_code ON customers(corporate_code) WHERE corporate_code IS NOT NULL;
CREATE INDEX ix_customers_document ON customers(document_number);
