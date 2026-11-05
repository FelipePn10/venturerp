-- Material em consignação. SAIDA: mercadoria nossa com o cliente, que segue no nosso
-- estoque (almoxarifado de consignação) até ele vender ou devolver. ENTRADA: material
-- do fornecedor conosco, que só vira estoque nosso quando for consumido.
ALTER TABLE stock_parameters
    ADD COLUMN consignment_warehouse_id bigint REFERENCES warehouse(id),
    ADD COLUMN consignment_days integer CHECK (consignment_days IS NULL OR consignment_days > 0);

CREATE TABLE stock_consignments (
    id                  bigserial      PRIMARY KEY,
    enterprise_id       bigint         NOT NULL REFERENCES enterprise(id),
    direction           varchar(7)     NOT NULL CHECK (direction IN ('SAIDA','ENTRADA')),
    partner_code        bigint         NOT NULL,
    item_code           bigint         NOT NULL,
    mask                varchar(200)   NOT NULL DEFAULT '',
    lot                 varchar(50),
    origin_warehouse_id bigint         REFERENCES warehouse(id),
    warehouse_id        bigint         REFERENCES warehouse(id),
    quantity            numeric(18,6)  NOT NULL CHECK (quantity > 0),
    settled_qty         numeric(18,6)  NOT NULL DEFAULT 0 CHECK (settled_qty >= 0),
    returned_qty        numeric(18,6)  NOT NULL DEFAULT 0 CHECK (returned_qty >= 0),
    unit_value          numeric(24,10) NOT NULL DEFAULT 0 CHECK (unit_value >= 0),
    fiscal_exit_id      bigint,
    document            varchar(60),
    issue_date          date           NOT NULL,
    due_date            date,
    status              varchar(10)    NOT NULL DEFAULT 'ABERTA' CHECK (status IN ('ABERTA','ENCERRADA','CANCELADA')),
    notes               text,
    created_by          uuid           NOT NULL REFERENCES users(id),
    created_at          timestamptz    NOT NULL DEFAULT NOW(),
    updated_at          timestamptz    NOT NULL DEFAULT NOW(),
    CHECK (settled_qty + returned_qty <= quantity),
    CHECK ((direction = 'SAIDA') = (warehouse_id IS NOT NULL))
);
CREATE INDEX idx_stock_consignments_abertas ON stock_consignments (enterprise_id, direction, partner_code, item_code, mask) WHERE status = 'ABERTA';
CREATE INDEX idx_stock_consignments_nota ON stock_consignments (enterprise_id, fiscal_exit_id) WHERE fiscal_exit_id IS NOT NULL;

CREATE TABLE stock_consignment_settlements (
    id             bigserial     PRIMARY KEY,
    consignment_id bigint        NOT NULL REFERENCES stock_consignments(id) ON DELETE CASCADE,
    kind           varchar(10)   NOT NULL CHECK (kind IN ('VENDA','CONSUMO','DEVOLUCAO')),
    quantity       numeric(18,6) NOT NULL CHECK (quantity > 0),
    settled_on     date          NOT NULL,
    fiscal_exit_id bigint,
    document       varchar(60),
    reversed       boolean       NOT NULL DEFAULT FALSE,
    created_by     uuid          NOT NULL REFERENCES users(id),
    created_at     timestamptz   NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_stock_consignment_settlements_nota ON stock_consignment_settlements (fiscal_exit_id) WHERE fiscal_exit_id IS NOT NULL;

-- Transferência entre estabelecimentos da mesma raiz de CNPJ. A nota sai da origem,
-- o material fica no trânsito da origem e só a entrada da nota no destino o baixa.
CREATE TABLE stock_cnpj_transfers (
    id                 bigserial   PRIMARY KEY,
    origin_id          bigint      NOT NULL REFERENCES enterprise(id),
    destination_id     bigint      NOT NULL REFERENCES enterprise(id),
    fiscal_exit_id     bigint      NOT NULL,
    nfe_key            varchar(44) NOT NULL,
    transit_warehouse  bigint      NOT NULL REFERENCES warehouse(id),
    status             varchar(11) NOT NULL DEFAULT 'EM_TRANSITO' CHECK (status IN ('EM_TRANSITO','RECEBIDA','CANCELADA')),
    fiscal_entry_id    bigint,
    received_at        timestamptz,
    created_by         uuid        NOT NULL REFERENCES users(id),
    created_at         timestamptz NOT NULL DEFAULT NOW(),
    CHECK (origin_id <> destination_id)
);
CREATE UNIQUE INDEX uq_stock_cnpj_transfers_nota ON stock_cnpj_transfers (origin_id, fiscal_exit_id);
CREATE INDEX idx_stock_cnpj_transfers_chave ON stock_cnpj_transfers (destination_id, nfe_key) WHERE status = 'EM_TRANSITO';

CREATE TABLE stock_cnpj_transfer_lines (
    id            bigserial     PRIMARY KEY,
    transfer_id   bigint        NOT NULL REFERENCES stock_cnpj_transfers(id) ON DELETE CASCADE,
    origin_item   bigint        NOT NULL,
    mask          varchar(200)  NOT NULL DEFAULT '',
    lot           varchar(50),
    from_warehouse bigint       NOT NULL REFERENCES warehouse(id),
    quantity      numeric(18,6) NOT NULL CHECK (quantity > 0)
);

SELECT instalar_conferencia_de_empresa();
