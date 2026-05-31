-- ─── Invoice Types & Tax Types ────────────────────────────────────────────────

CREATE TYPE invoice_type_enum AS ENUM (
    'VENDA',
    'DEVOLUCAO',
    'REMESSA',
    'REMESSA_CONSIGNACAO',
    'REMESSA_ARMAZENAGEM',
    'REMESSA_BENEFICIAMENTO',
    'RETORNO_BENEFICIAMENTO',
    'SIMPLES_REMESSA',
    'TRANSFERENCIA',
    'VENDA_CONSIGNACAO',
    'COMPLEMENTAR_ICM',
    'COMPLEMENTAR_IPI',
    'DEMONSTRACAO',
    'EMPRESTIMO',
    'FATURAMENTO_ANTECIPADO',
    'PRESTACAO_SERVICOS',
    'OUTROS'
);

CREATE TYPE invoice_stock_enum AS ENUM (
    'ATUALIZA',
    'NAO_ATUALIZA',
    'TRANSFERENCIA_EXTERNA'
);

CREATE TYPE invoice_icms_type_enum AS ENUM ('TRIBUTADO', 'ISENTO', 'OUTROS');

CREATE TABLE invoice_types (
    id                          BIGSERIAL PRIMARY KEY,
    code                        BIGINT               NOT NULL UNIQUE,
    description                 VARCHAR(200)         NOT NULL,
    type                        invoice_type_enum    NOT NULL DEFAULT 'VENDA',
    stock_movement              invoice_stock_enum   NOT NULL DEFAULT 'ATUALIZA',
    icms_type                   invoice_icms_type_enum NOT NULL DEFAULT 'TRIBUTADO',
    -- Fiscal percentages
    icms_pct                    NUMERIC(5,2)         NOT NULL DEFAULT 0,
    icms_reduction_pct          NUMERIC(5,2)         NOT NULL DEFAULT 0,
    ipi_pct                     NUMERIC(5,2)         NOT NULL DEFAULT 0,
    pis_pct                     NUMERIC(5,2)         NOT NULL DEFAULT 0,
    cofins_pct                  NUMERIC(5,2)         NOT NULL DEFAULT 0,
    issqn_pct                   NUMERIC(5,2)         NOT NULL DEFAULT 0,
    ir_pct                      NUMERIC(5,2)         NOT NULL DEFAULT 0,
    csll_pct                    NUMERIC(5,2)         NOT NULL DEFAULT 0,
    inss_pct                    NUMERIC(5,2)         NOT NULL DEFAULT 0,
    -- Behavioral flags
    generates_revenue           BOOLEAN              NOT NULL DEFAULT TRUE,
    updates_inventory           BOOLEAN              NOT NULL DEFAULT TRUE,
    generates_financial_title   BOOLEAN              NOT NULL DEFAULT TRUE,
    considers_goals             BOOLEAN              NOT NULL DEFAULT FALSE,
    calc_substitution_tax       BOOLEAN              NOT NULL DEFAULT FALSE,
    calc_icms_deferral          BOOLEAN              NOT NULL DEFAULT FALSE,
    calc_pis_cofins             BOOLEAN              NOT NULL DEFAULT FALSE,
    calc_difal                  BOOLEAN              NOT NULL DEFAULT TRUE,
    requires_sales_order        BOOLEAN              NOT NULL DEFAULT FALSE,
    lists_fiscal_books          BOOLEAN              NOT NULL DEFAULT TRUE,
    is_active                   BOOLEAN              NOT NULL DEFAULT TRUE,
    created_at                  TIMESTAMPTZ          NOT NULL DEFAULT NOW()
);

-- Tax Types (base calculation parameters for taxes on outbound invoices)
CREATE TABLE tax_types (
    id                          BIGSERIAL PRIMARY KEY,
    code                        BIGINT       NOT NULL UNIQUE,
    description                 VARCHAR(150) NOT NULL,
    -- IPI base includes
    ipi_base_total_items        BOOLEAN      NOT NULL DEFAULT TRUE,
    ipi_base_subtract_discount  BOOLEAN      NOT NULL DEFAULT FALSE,
    ipi_base_add_freight        BOOLEAN      NOT NULL DEFAULT FALSE,
    ipi_base_add_expenses       BOOLEAN      NOT NULL DEFAULT FALSE,
    -- ICMS base includes
    icms_base_total_items       BOOLEAN      NOT NULL DEFAULT TRUE,
    icms_base_subtract_discount BOOLEAN      NOT NULL DEFAULT TRUE,
    icms_base_add_freight       BOOLEAN      NOT NULL DEFAULT TRUE,
    icms_base_add_ipi           BOOLEAN      NOT NULL DEFAULT FALSE,
    icms_base_add_expenses      BOOLEAN      NOT NULL DEFAULT FALSE,
    -- PIS/COFINS base includes
    pis_cofins_base_total_items         BOOLEAN NOT NULL DEFAULT TRUE,
    pis_cofins_base_subtract_discount   BOOLEAN NOT NULL DEFAULT TRUE,
    pis_cofins_base_add_freight         BOOLEAN NOT NULL DEFAULT FALSE,
    pis_cofins_base_add_insurance       BOOLEAN NOT NULL DEFAULT FALSE,
    pis_cofins_base_add_expenses        BOOLEAN NOT NULL DEFAULT FALSE,
    -- CSLL base includes
    csll_base_total_items        BOOLEAN     NOT NULL DEFAULT TRUE,
    csll_base_subtract_discount  BOOLEAN     NOT NULL DEFAULT TRUE,
    csll_base_add_freight        BOOLEAN     NOT NULL DEFAULT FALSE,
    -- IR base includes
    ir_base_total_items          BOOLEAN     NOT NULL DEFAULT TRUE,
    ir_base_subtract_discount    BOOLEAN     NOT NULL DEFAULT TRUE,
    ir_base_add_freight          BOOLEAN     NOT NULL DEFAULT FALSE,
    -- Consumer indicator
    is_consumer                  BOOLEAN     NOT NULL DEFAULT FALSE,
    is_active                    BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
