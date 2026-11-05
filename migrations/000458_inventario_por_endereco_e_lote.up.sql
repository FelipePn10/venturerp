-- Fase A da evolução de estoque (.ai/tasks/estoque-suprimentos-evolucao.md).
--
-- O inventário fotografava o item no almoxarifado: a diferença não sabia de qual
-- endereço nem de qual lote era. Agora cada linha é item + endereço + lote, como
-- o agrupamento detalhado do Focco. Entram também as opções de condução:
-- bloquear a movimentação durante a contagem, contagem cega, recontagem e
-- aprovação do ajuste por tolerância em % e em valor.

ALTER TABLE physical_inventories
    ADD COLUMN blocks_movements BOOLEAN       NOT NULL DEFAULT FALSE,
    ADD COLUMN blind_count      BOOLEAN       NOT NULL DEFAULT FALSE,
    ADD COLUMN partial          BOOLEAN       NOT NULL DEFAULT FALSE,
    ADD COLUMN approval_mode    VARCHAR(20)   NOT NULL DEFAULT 'NUNCA'
        CHECK (approval_mode IN ('NUNCA', 'FORA_TOLERANCIA', 'SEMPRE')),
    ADD COLUMN tolerance_pct    NUMERIC(7,4)  CHECK (tolerance_pct IS NULL OR tolerance_pct >= 0),
    ADD COLUMN tolerance_value  NUMERIC(15,2) CHECK (tolerance_value IS NULL OR tolerance_value >= 0),
    ADD COLUMN close_reason     TEXT,
    ADD COLUMN zero_uncounted   BOOLEAN       NOT NULL DEFAULT FALSE;

ALTER TABLE physical_inventory_items
    ADD COLUMN address       VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN lot           VARCHAR(50)  NOT NULL DEFAULT '',
    ADD COLUMN needs_recount BOOLEAN      NOT NULL DEFAULT FALSE;

ALTER TABLE physical_inventory_items DROP CONSTRAINT IF EXISTS uq_physical_inventory_items_item;
DROP INDEX IF EXISTS uq_physical_inventory_items_item;
CREATE UNIQUE INDEX uq_physical_inventory_items_item
    ON physical_inventory_items (inventory_id, item_code, mask, warehouse_id, address, lot);

-- Um almoxarifado não tem dois inventários em andamento. A regra estava só no
-- código, com uma janela entre a leitura e a gravação.
CREATE UNIQUE INDEX uq_physical_inventories_em_andamento
    ON physical_inventories (enterprise_id, warehouse_id)
    WHERE status IN ('OPEN', 'EM_CONFERENCIA', 'AGUARDANDO_APROVACAO');

ALTER TABLE aprovacao_politicas DROP CONSTRAINT IF EXISTS aprovacao_politicas_tipo_check;
ALTER TABLE aprovacao_politicas ADD CONSTRAINT aprovacao_politicas_tipo_check
    CHECK (tipo IN ('LIBERACAO_CREDITO', 'ORCAMENTO_POLITICA_COMERCIAL', 'PEDIDO_POLITICA_COMERCIAL', 'AJUSTE_INVENTARIO'));

-- Acerto de inconsistência de saldo: os movimentos são a verdade e o saldo é
-- refeito a partir deles. Só cresce: é o rastro do que foi mudado sem movimento.
CREATE TABLE stock_inconsistency_fixes (
    id            BIGSERIAL     PRIMARY KEY,
    enterprise_id BIGINT        NOT NULL REFERENCES enterprise(id),
    kind          VARCHAR(30)   NOT NULL,
    item_code     BIGINT        NOT NULL,
    mask          VARCHAR(200)  NOT NULL DEFAULT '',
    warehouse_id  BIGINT        NOT NULL,
    before_qty    NUMERIC(20,6) NOT NULL,
    after_qty     NUMERIC(20,6) NOT NULL,
    reason        TEXT          NOT NULL,
    fixed_by      UUID          NOT NULL REFERENCES users(id),
    fixed_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_stock_inconsistency_fixes_tenant ON stock_inconsistency_fixes (enterprise_id, item_code);

SELECT instalar_conferencia_de_empresa();
