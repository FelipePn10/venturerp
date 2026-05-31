BEGIN;

-- ─── Supplier Types (Cadastro de Tipos de Fornecedor) ─────────────────────────
-- kind drives the "Inscrição Estadual" obligation rule:
--   NORMAL              → IE required
--   TRANSPORTADORA      → IE optional
--   TRANSP_REDESP       → IE optional
--   REDESPACHO          → IE optional
CREATE TABLE IF NOT EXISTS supplier_types (
    id          BIGSERIAL PRIMARY KEY,
    code        BIGINT       NOT NULL UNIQUE,
    description VARCHAR(150) NOT NULL,
    kind        VARCHAR(20)  NOT NULL DEFAULT 'NORMAL'
                CHECK (kind IN ('NORMAL', 'TRANSPORTADORA', 'TRANSP_REDESP', 'REDESPACHO')),
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ─── Supplier Contact Types (FCLI0103 FOR) ────────────────────────────────────
CREATE TABLE IF NOT EXISTS supplier_contact_types (
    id          BIGSERIAL PRIMARY KEY,
    code        BIGINT       NOT NULL UNIQUE,
    description VARCHAR(150) NOT NULL,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ─── Suppliers (Cadastro de Fornecedores) ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS suppliers (
    id                                BIGSERIAL PRIMARY KEY,
    -- Identification
    code                              BIGINT       NOT NULL UNIQUE,
    corporate_code                    BIGINT,                       -- Código Pai (grouping)
    is_active                         BOOLEAN      NOT NULL DEFAULT TRUE,
    is_representative                 BOOLEAN      NOT NULL DEFAULT FALSE,
    is_customer                       BOOLEAN      NOT NULL DEFAULT FALSE,
    name                              VARCHAR(200) NOT NULL,        -- Descrição / razão social
    trade_name                        VARCHAR(200),                 -- Fantasia
    -- Person / documents
    person_type                       VARCHAR(10)  NOT NULL DEFAULT 'JURIDICA'
                                      CHECK (person_type IN ('JURIDICA', 'FISICA')),
    document_type                     VARCHAR(20)  NOT NULL DEFAULT 'CNPJ'
                                      CHECK (document_type IN ('CNPJ', 'CPF', 'ESTRANGEIRO', 'ISENTO')),
    document_number                   VARCHAR(20)  NOT NULL,        -- CNPJ / CPF
    state_registration                VARCHAR(30),                  -- Inscr. Est.
    municipal_registration            VARCHAR(30),                  -- Insc. Mun.
    -- Classification / commercial (reuse shared registers)
    supplier_type_id                  BIGINT       REFERENCES supplier_types(id),
    payment_condition_id              BIGINT       REFERENCES payment_conditions(id),
    carrier_id                        BIGINT       REFERENCES carriers(id),
    region_id                         BIGINT       REFERENCES regions(id),
    freight_type                      VARCHAR(15)  NOT NULL DEFAULT 'SEM_FRETE'
                                      CHECK (freight_type IN ('CIF','DAF','FOB','SEM_FRETE','CONVENIO','RETIRA','CORTESIA','TERCEIROS')),
    register_date                     DATE         NOT NULL DEFAULT CURRENT_DATE, -- Dt. Cadastro
    -- Fiscal flags
    viticola_obligation               VARCHAR(10)  NOT NULL DEFAULT 'NUNCA'
                                      CHECK (viticola_obligation IN ('NUNCA','AS_VEZES','SEMPRE')),
    gln_code                          VARCHAR(13),                  -- Código GLN
    agriculture_ministry_registration VARCHAR(11),                  -- Registro M.A. (AA-99999-9)
    icms_contributor                  VARCHAR(20)  NOT NULL DEFAULT 'CONTRIBUINTE'
                                      CHECK (icms_contributor IN ('CONTRIBUINTE','NAO_CONTRIBUINTE','ISENTO')),
    is_mei                            BOOLEAN      NOT NULL DEFAULT FALSE, -- Microempreendedor Individual
    tracking_platform                 VARCHAR(20)  NOT NULL DEFAULT 'NENHUM'
                                      CHECK (tracking_platform IN ('SSW','FRETEWEB','ENGLOBA_SISTEMAS','NENHUM')),
    homologated                       BOOLEAN      NOT NULL DEFAULT FALSE,
    -- SEFAZ consultation snapshot
    last_sefaz_query                  DATE,
    billing_receipt_status            VARCHAR(10)  CHECK (billing_receipt_status IN ('LIBERADO','BLOQUEADO')),
    last_sefaz_update                 DATE,
    sefaz_update_user                 VARCHAR(150),
    -- Status / audit
    blocked                           BOOLEAN      NOT NULL DEFAULT FALSE,
    block_reason                      TEXT,
    created_at                        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by                        UUID         NOT NULL,
    updated_at                        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_suppliers_corporate_code ON suppliers(corporate_code) WHERE corporate_code IS NOT NULL;
CREATE INDEX IF NOT EXISTS ix_suppliers_document ON suppliers(document_number);

-- ─── Supplier Addresses ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS supplier_addresses (
    id           BIGSERIAL PRIMARY KEY,
    supplier_id  BIGINT       NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    address_type VARCHAR(15)  NOT NULL DEFAULT 'COMERCIAL'
                 CHECK (address_type IN ('COBRANCA','ENTREGA','COMERCIAL','OUTRO')),
    zip_code     VARCHAR(10),                                    -- Cep
    street       VARCHAR(200),                                   -- Logradouro
    number       VARCHAR(20),                                    -- Nº
    complement   VARCHAR(100),                                   -- Compl.
    neighborhood VARCHAR(100),                                   -- Bairro
    city         VARCHAR(100),                                   -- Cidade
    uf           CHAR(2),                                        -- UF
    country      VARCHAR(60)  NOT NULL DEFAULT 'Brasil',
    is_default   BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_supplier_default_address
    ON supplier_addresses(supplier_id, address_type)
    WHERE is_default = TRUE;

-- ─── Supplier Phones (Pasta Telefones) ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS supplier_phones (
    id          BIGSERIAL PRIMARY KEY,
    supplier_id BIGINT      NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    number      VARCHAR(30) NOT NULL,
    ranking     INT         NOT NULL DEFAULT 1,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Supplier Emails (Pasta E-mails) ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS supplier_emails (
    id          BIGSERIAL PRIMARY KEY,
    supplier_id BIGINT       NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    email       VARCHAR(200) NOT NULL,
    ranking     INT          NOT NULL DEFAULT 1,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ─── Supplier Due Dates (Pasta Vencimentos / Inf. Recebimento) ────────────────
CREATE TABLE IF NOT EXISTS supplier_due_dates (
    id                   BIGSERIAL PRIMARY KEY,
    supplier_id          BIGINT       NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    description          VARCHAR(150) NOT NULL,
    ranking              INT          NOT NULL DEFAULT 1,
    base_date            VARCHAR(15)  NOT NULL DEFAULT 'EMISSAO'
                         CHECK (base_date IN ('EMISSAO','ENTRADA','DIGITACAO')),
    payment_condition_id BIGINT       REFERENCES payment_conditions(id),
    payment_type         VARCHAR(15)  NOT NULL DEFAULT 'NAO_INFORMADO'
                         CHECK (payment_type IN ('SEMANAL','MENSAL','NAO_INFORMADO')),
    subsequent_month     BOOLEAN      NOT NULL DEFAULT FALSE,
    rounding             VARCHAR(15)  NOT NULL DEFAULT 'FIXO'
                         CHECK (rounding IN ('POSTERGA','ANTECIPA','UTIL','FIXO')),
    receipt_start_time   VARCHAR(5),                              -- HH:MM
    receipt_end_time     VARCHAR(5),                              -- HH:MM
    avg_unload_minutes   INT,                                     -- Tempo Médio Descarga
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ─── Supplier Contacts (Pasta Contatos) ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS supplier_contacts (
    id                 BIGSERIAL PRIMARY KEY,
    supplier_id        BIGINT       NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    contact_type_id    BIGINT       REFERENCES supplier_contact_types(id),
    name               VARCHAR(150) NOT NULL,
    position           VARCHAR(100),                              -- Cargo
    department         VARCHAR(100),                              -- Departamento
    ranking            INT          NOT NULL DEFAULT 1,
    observation        TEXT,                                      -- Obs./Outros
    purchase_order_tag VARCHAR(100),                              -- Tag do Pedido de Compra
    is_active          BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ─── Supplier Contact Phones / E-mails ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS supplier_contact_phones (
    id         BIGSERIAL PRIMARY KEY,
    contact_id BIGINT      NOT NULL REFERENCES supplier_contacts(id) ON DELETE CASCADE,
    value      VARCHAR(30) NOT NULL,
    ranking    INT         NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS supplier_contact_emails (
    id         BIGSERIAL PRIMARY KEY,
    contact_id BIGINT       NOT NULL REFERENCES supplier_contacts(id) ON DELETE CASCADE,
    value      VARCHAR(200) NOT NULL,
    ranking    INT          NOT NULL DEFAULT 1
);

-- ─── Supplier ↔ Enterprise link (Pasta Empresas) ──────────────────────────────
-- Per-enterprise fiscal/financial binding: financial account, IPI flag,
-- default invoice type used on entry NF, purchase price table.
CREATE TABLE IF NOT EXISTS supplier_enterprises (
    id                      BIGSERIAL PRIMARY KEY,
    supplier_id             BIGINT       NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    enterprise_code         BIGINT       NOT NULL,
    financial_account       VARCHAR(30),                          -- Conta do Planejamento Financeiro
    applies_ipi             BOOLEAN      NOT NULL DEFAULT FALSE,   -- checkbox IPI (pasta Empresa)
    default_invoice_type_id BIGINT       REFERENCES invoice_types(id),
    purchase_price_table_id BIGINT,
    is_active               BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (supplier_id, enterprise_code)
);

-- ─── Supplier Parameters (Parâmetros de Fornecedores, 1 por empresa) ──────────
CREATE TABLE IF NOT EXISTS supplier_parameters (
    id                            BIGSERIAL PRIMARY KEY,
    enterprise_code               BIGINT      NOT NULL UNIQUE,
    default_financial_account     VARCHAR(30),                    -- P1
    unique_item_code_per_supplier BOOLEAN     NOT NULL DEFAULT FALSE, -- P2
    requires_financial_account    BOOLEAN     NOT NULL DEFAULT FALSE, -- P3
    purchase_supplier_type_id     BIGINT      REFERENCES supplier_types(id), -- P4
    copy_obs_to_purchase_order    BOOLEAN     NOT NULL DEFAULT FALSE, -- P5
    copy_obs_to_entry_invoice     BOOLEAN     NOT NULL DEFAULT FALSE, -- P6
    homologation_default          BOOLEAN     NOT NULL DEFAULT FALSE, -- P7
    use_stock_uom                 BOOLEAN     NOT NULL DEFAULT FALSE, -- P8
    generic_supplier_code         BIGINT,                         -- P9
    default_due_base_date         VARCHAR(15) NOT NULL DEFAULT 'EMISSAO' -- P10
                                  CHECK (default_due_base_date IN ('EMISSAO','ENTRADA','DIGITACAO')),
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Close the purchase_orders.supplier_code link ─────────────────────────────
ALTER TABLE public.purchase_orders
    ADD CONSTRAINT fk_purchase_orders_supplier
    FOREIGN KEY (supplier_code) REFERENCES suppliers(code);

COMMIT;
