-- Estoque, fase 2: estorno de movimento, tipo de movimento configurado no
-- lançamento, centro de custo no movimento e requisição ao almoxarifado.

-- Estorno: o movimento estornado aponta para o estorno e vice-versa, e um
-- movimento só é estornado uma vez.
ALTER TABLE stock_movements
    ADD COLUMN IF NOT EXISTS reverses_movement_id    BIGINT REFERENCES stock_movements(id),
    ADD COLUMN IF NOT EXISTS reversed_by_movement_id BIGINT REFERENCES stock_movements(id),
    -- Sigla do tipo de movimento do cadastro (VEST0200) usado no lançamento.
    ADD COLUMN IF NOT EXISTS movement_type_code      VARCHAR(20),
    -- Centro de custo da saída (requisição, consumo interno).
    ADD COLUMN IF NOT EXISTS cost_center_code        INT;
CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_movements_reverses
    ON stock_movements (reverses_movement_id) WHERE reverses_movement_id IS NOT NULL;

-- O cadastro de tipos de movimento passa a governar o lançamento manual.
ALTER TABLE stock_movement_types
    ADD COLUMN IF NOT EXISTS requires_cost_center BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS allows_manual        BOOLEAN NOT NULL DEFAULT TRUE;

-- Requisição ao almoxarifado: quem pede, para qual centro de custo, o quê; o
-- almoxarife atende (total ou parcialmente) e cada atendimento é uma saída.
CREATE TABLE IF NOT EXISTS material_requisitions (
    id                      BIGSERIAL PRIMARY KEY,
    enterprise_id           BIGINT NOT NULL REFERENCES enterprise(id),
    code                    BIGINT NOT NULL,
    warehouse_id            BIGINT NOT NULL REFERENCES warehouse(id),
    requester_name          VARCHAR(120),
    requester_employee_code BIGINT,
    cost_center_code        INT,
    needed_date             DATE,
    status                  VARCHAR(20) NOT NULL DEFAULT 'ABERTA'
                            CHECK (status IN ('ABERTA','PARCIAL','ATENDIDA','CANCELADA')),
    notes                   TEXT,
    created_by              UUID NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, code)
);
CREATE INDEX IF NOT EXISTS ix_material_requisitions_status ON material_requisitions (enterprise_id, status);

CREATE TABLE IF NOT EXISTS material_requisition_items (
    id             BIGSERIAL PRIMARY KEY,
    requisition_id BIGINT NOT NULL REFERENCES material_requisitions(id) ON DELETE CASCADE,
    sequence       INT NOT NULL,
    item_code      BIGINT NOT NULL,
    mask           VARCHAR(200) NOT NULL DEFAULT '',
    quantity       NUMERIC(20,6) NOT NULL CHECK (quantity > 0),
    issued_qty     NUMERIC(20,6) NOT NULL DEFAULT 0 CHECK (issued_qty >= 0),
    notes          TEXT,
    UNIQUE (requisition_id, sequence)
);

SELECT instalar_conferencia_de_empresa();
