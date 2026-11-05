-- Fase C da evolução de estoque (.ai/tasks/estoque-suprimentos-evolucao.md).
--
-- O movimento ganha a data do documento (até aqui só existia a hora da gravação)
-- e o estoque ganha fechamento mensal: foto de saldo e custo, recálculo de quem
-- teve lançamento retroativo e diferença de custo levada à contabilidade.

ALTER TABLE stock_movements
    ADD COLUMN movement_date DATE,
    ADD COLUMN backdated     BOOLEAN NOT NULL DEFAULT FALSE;

-- A data vem da hora da gravação, no fuso do Brasil. Sem disparar a auditoria:
-- não é uma alteração de movimento.
ALTER TABLE stock_movements DISABLE TRIGGER trg_stock_movements_inventory_audit;
UPDATE stock_movements SET movement_date = (created_at AT TIME ZONE 'America/Sao_Paulo')::date;
ALTER TABLE stock_movements ENABLE TRIGGER trg_stock_movements_inventory_audit;

ALTER TABLE stock_movements
    ALTER COLUMN movement_date SET NOT NULL,
    ALTER COLUMN movement_date SET DEFAULT CURRENT_DATE;

CREATE INDEX idx_stock_movements_data ON stock_movements (enterprise_id, item_code, movement_date, id);
CREATE INDEX idx_stock_movements_retroativos ON stock_movements (enterprise_id, item_code) WHERE backdated;

CREATE TABLE stock_closings (
    id                 BIGSERIAL     PRIMARY KEY,
    enterprise_id      BIGINT        NOT NULL REFERENCES enterprise(id),
    competence         DATE          NOT NULL,
    status             VARCHAR(10)   NOT NULL DEFAULT 'FECHADO' CHECK (status IN ('FECHADO', 'REABERTO')),
    total_value        NUMERIC(18,2) NOT NULL DEFAULT 0,
    recalculated_items INTEGER       NOT NULL DEFAULT 0,
    variance_value     NUMERIC(18,2) NOT NULL DEFAULT 0,
    journal_posted     BOOLEAN       NOT NULL DEFAULT FALSE,
    closed_by          UUID          NOT NULL REFERENCES users(id),
    closed_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    reopened_by        UUID          REFERENCES users(id),
    reopened_at        TIMESTAMPTZ,
    reopen_reason      TEXT
);
CREATE UNIQUE INDEX uq_stock_closings_competencia ON stock_closings (enterprise_id, competence) WHERE status = 'FECHADO';

-- Saldo inicial do mês seguinte: quantidade e valor no fim da competência.
CREATE TABLE stock_closing_balances (
    closing_id   BIGINT        NOT NULL REFERENCES stock_closings(id) ON DELETE CASCADE,
    item_code    BIGINT        NOT NULL,
    mask         VARCHAR(200)  NOT NULL DEFAULT '',
    warehouse_id BIGINT        NOT NULL,
    quantity     NUMERIC(20,6) NOT NULL,
    avg_cost     NUMERIC(24,10) NOT NULL,
    total_value  NUMERIC(18,4) NOT NULL,
    PRIMARY KEY (closing_id, item_code, mask, warehouse_id)
);

-- Diferença de custo que não ficou no estoque: vai para o resultado no fechamento.
CREATE TABLE stock_cost_variances (
    id            BIGSERIAL     PRIMARY KEY,
    enterprise_id BIGINT        NOT NULL REFERENCES enterprise(id),
    kind          VARCHAR(20)   NOT NULL CHECK (kind IN ('SALDO_NEGATIVO', 'RECALCULO')),
    item_code     BIGINT        NOT NULL,
    mask          VARCHAR(200)  NOT NULL DEFAULT '',
    warehouse_id  BIGINT        NOT NULL,
    movement_id   BIGINT,
    value         NUMERIC(18,4) NOT NULL,
    occurred_on   DATE          NOT NULL,
    closing_id    BIGINT        REFERENCES stock_closings(id),
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_stock_cost_variances_aberto ON stock_cost_variances (enterprise_id, occurred_on) WHERE closing_id IS NULL;

-- O que o recálculo mudou em cada movimento: o valor antigo não se perde.
CREATE TABLE stock_cost_recalc_log (
    id          BIGSERIAL      PRIMARY KEY,
    closing_id  BIGINT         NOT NULL REFERENCES stock_closings(id) ON DELETE CASCADE,
    movement_id BIGINT         NOT NULL,
    old_unit    NUMERIC(24,10) NOT NULL,
    new_unit    NUMERIC(24,10) NOT NULL,
    old_total   NUMERIC(18,4)  NOT NULL,
    new_total   NUMERIC(18,4)  NOT NULL
);

SELECT instalar_conferencia_de_empresa();
