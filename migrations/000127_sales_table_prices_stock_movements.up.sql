-- Cadastro de Preços da Tabela de Vendas
CREATE TYPE price_situation_enum AS ENUM ('ATIVO', 'INATIVO', 'PROMOCIONAL');

CREATE TABLE sales_table_prices (
    id                  BIGSERIAL PRIMARY KEY,
    sales_table_id      BIGINT NOT NULL REFERENCES sales_tables(id),
    item_code           VARCHAR(60) NOT NULL,
    price               NUMERIC(15,4) NOT NULL DEFAULT 0,
    ume                 VARCHAR(6),
    umc                 VARCHAR(6),
    price_conv          NUMERIC(15,4) NOT NULL DEFAULT 0,
    formula             VARCHAR(100),
    situation           price_situation_enum NOT NULL DEFAULT 'ATIVO',
    blocked             BOOLEAN NOT NULL DEFAULT FALSE,
    observation         TEXT,
    product_line_id     BIGINT,
    item_mask           VARCHAR(60),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (sales_table_id, item_code)
);

-- Cadastro de Tipos de Movimentos de Estoque
CREATE TYPE stock_movement_usage_enum AS ENUM ('PRODUCAO', 'COMPRAS', 'VENDAS', 'GERAL', 'AJUSTE', 'TRANSFERENCIA');
CREATE TYPE stock_movement_direction_enum AS ENUM ('ENTRADA', 'SAIDA', 'TRANSFERENCIA', 'AMBOS');

CREATE TABLE stock_movement_types (
    id                      BIGSERIAL PRIMARY KEY,
    sigla                   VARCHAR(10) NOT NULL UNIQUE,
    description             VARCHAR(150) NOT NULL,
    usage_type              stock_movement_usage_enum NOT NULL DEFAULT 'GERAL',
    entry_order             BOOLEAN NOT NULL DEFAULT FALSE,
    exit_order              BOOLEAN NOT NULL DEFAULT FALSE,
    considers_consumption   BOOLEAN NOT NULL DEFAULT FALSE,
    updates_avg_cost        BOOLEAN NOT NULL DEFAULT FALSE,
    is_adjustment           BOOLEAN NOT NULL DEFAULT FALSE,
    updates_cycle_count     BOOLEAN NOT NULL DEFAULT FALSE,
    shows_in_summary        BOOLEAN NOT NULL DEFAULT TRUE,
    entry_exit              stock_movement_direction_enum NOT NULL DEFAULT 'AMBOS',
    generates_fci_movement  BOOLEAN NOT NULL DEFAULT FALSE,
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
