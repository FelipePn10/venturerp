-- Fase B da evolução de estoque (.ai/tasks/estoque-suprimentos-evolucao.md).
--
-- Lote passa a ter situação, grau e ação ao vencer; o item diz a regra de baixa
-- (FEFO ou FIFO); a linha da NF-e de saída guarda máscara, lote e endereço.

ALTER TABLE manufacturing_stock_item_controls
    ADD COLUMN lot_issue_rule VARCHAR(4) NOT NULL DEFAULT 'FEFO' CHECK (lot_issue_rule IN ('FEFO', 'FIFO'));

-- Situação de lote é cadastro da empresa: cada uma diz o que o lote pode fazer.
-- Almoxarifado, endereço e série já têm controle próprio.
CREATE TABLE stock_lot_statuses (
    enterprise_id    BIGINT       NOT NULL REFERENCES enterprise(id),
    code             VARCHAR(20)  NOT NULL,
    description      VARCHAR(120) NOT NULL,
    allows_issue     BOOLEAN      NOT NULL DEFAULT TRUE,
    allows_transfer  BOOLEAN      NOT NULL DEFAULT TRUE,
    counts_available BOOLEAN      NOT NULL DEFAULT TRUE,
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_id, code)
);

ALTER TABLE stock_lots
    ADD COLUMN status_code VARCHAR(20),
    ADD COLUMN grade       VARCHAR(20),
    ADD CONSTRAINT fk_stock_lots_status FOREIGN KEY (enterprise_id, status_code)
        REFERENCES stock_lot_statuses (enterprise_id, code);

-- Toda troca de situação fica registrada: é o que explica por que um lote parou.
CREATE TABLE stock_lot_status_events (
    id            BIGSERIAL    PRIMARY KEY,
    enterprise_id BIGINT       NOT NULL REFERENCES enterprise(id),
    item_code     BIGINT       NOT NULL,
    mask          VARCHAR(200) NOT NULL DEFAULT '',
    lot           VARCHAR(50)  NOT NULL,
    from_status   VARCHAR(20),
    to_status     VARCHAR(20),
    reason        TEXT         NOT NULL,
    changed_by    UUID         NOT NULL REFERENCES users(id),
    changed_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_stock_lot_status_events ON stock_lot_status_events (enterprise_id, item_code, lot);

-- Parâmetros do estoque, um registro por empresa. Sem registro vale o padrão.
CREATE TABLE stock_parameters (
    enterprise_id      BIGINT      PRIMARY KEY REFERENCES enterprise(id),
    expired_lot_action VARCHAR(10) NOT NULL DEFAULT 'BLOQUEAR' CHECK (expired_lot_action IN ('BLOQUEAR', 'AVISAR')),
    updated_by         UUID        REFERENCES users(id),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE fiscal_exit_items
    ADD COLUMN mask    VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN lot     VARCHAR(50),
    ADD COLUMN address VARCHAR(100);

SELECT instalar_conferencia_de_empresa();
