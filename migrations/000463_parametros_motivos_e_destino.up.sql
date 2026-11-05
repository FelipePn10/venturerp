-- Regras de estoque que antes vinham da tela ou não existiam.
ALTER TABLE stock_parameters
    ADD COLUMN retroactive_months smallint CHECK (retroactive_months IS NULL OR retroactive_months >= 0),
    ADD COLUMN tolerance_a numeric(7,4) CHECK (tolerance_a IS NULL OR tolerance_a >= 0),
    ADD COLUMN tolerance_b numeric(7,4) CHECK (tolerance_b IS NULL OR tolerance_b >= 0),
    ADD COLUMN tolerance_c numeric(7,4) CHECK (tolerance_c IS NULL OR tolerance_c >= 0),
    ADD COLUMN obsolete_days integer CHECK (obsolete_days IS NULL OR obsolete_days > 0),
    ADD COLUMN obsolete_provision_pct numeric(7,4) CHECK (obsolete_provision_pct IS NULL OR obsolete_provision_pct BETWEEN 0 AND 100);

-- Motivos de divergência: cada empresa cadastra os seus.
CREATE TABLE stock_discrepancy_reasons (
    enterprise_id bigint NOT NULL REFERENCES enterprise(id),
    code varchar(20) NOT NULL,
    description varchar(120) NOT NULL,
    is_active boolean NOT NULL DEFAULT TRUE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_id, code)
);

-- Para qual almoxarifado a requisição de compra foi pedida.
ALTER TABLE purchase_requisition_items ADD COLUMN warehouse_id bigint;

-- Quanto da separação da onda a nota fiscal já levou: a nota sai no lote e no
-- endereço separados.
ALTER TABLE stock_picking_wave_lines ADD COLUMN invoiced_qty numeric(18,6) NOT NULL DEFAULT 0;

-- Almoxarifado que o cartão kanban abastece.
ALTER TABLE kanban_cards ADD COLUMN warehouse_id bigint;

-- De qual linha da onda a saída da nota saiu: o cancelamento devolve a separação.
ALTER TABLE stock_movements ADD COLUMN wave_line_id bigint;

-- A prioridade calculada pelo MRP nunca teve onde ser gravada: a atualização falhava.
ALTER TABLE mrp_planned_suggestions ADD COLUMN priority varchar(20);
