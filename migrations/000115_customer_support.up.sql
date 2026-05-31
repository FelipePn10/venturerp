-- ─── Customer Support Tables ─────────────────────────────────────────────────

-- Regions
CREATE TABLE regions (
    id          BIGSERIAL PRIMARY KEY,
    code        BIGINT      NOT NULL UNIQUE,
    description VARCHAR(100) NOT NULL,
    uf          CHAR(2),
    city        VARCHAR(100),
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  UUID        NOT NULL
);

-- Market Segments (self-referencing for hierarchy)
CREATE TABLE market_segments (
    id                   BIGSERIAL PRIMARY KEY,
    code                 BIGINT       NOT NULL UNIQUE,
    description          VARCHAR(150) NOT NULL,
    parent_id            BIGINT       REFERENCES market_segments(id),
    has_pis_cofins_retention BOOLEAN NOT NULL DEFAULT FALSE,
    retention_indicator  SMALLINT,
    is_active            BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Customer Contact Types
CREATE TABLE customer_contact_types (
    id          BIGSERIAL PRIMARY KEY,
    code        BIGINT       NOT NULL UNIQUE,
    description VARCHAR(100) NOT NULL,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Customer Types
CREATE TYPE customer_category_enum AS ENUM ('NORMAL', 'CONSUMIDOR');

CREATE TABLE customer_types (
    id            BIGSERIAL PRIMARY KEY,
    code          BIGINT                NOT NULL UNIQUE,
    description   VARCHAR(150)          NOT NULL UNIQUE,
    category      customer_category_enum NOT NULL DEFAULT 'NORMAL',
    delivery_days SMALLINT              NOT NULL DEFAULT 0,
    is_active     BOOLEAN               NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ           NOT NULL DEFAULT NOW()
);
