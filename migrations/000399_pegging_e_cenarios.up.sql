-- Pegging (de onde vem cada necessidade) e cenários de simulação do
-- planejamento. Ver .ai/tasks/pegging-e-cenarios.md.

-- ── Pegging ────────────────────────────────────────────────────────────────
-- Gravado a cada cálculo do MRP (pegging fixo, como no SAP MD04): cada linha
-- diz que parte de uma necessidade foi coberta por qual suprimento. A
-- necessidade de um componente aponta a sugestão do pai (demand_kind =
-- 'SUGESTAO'); seguindo essas pontes chega-se do pedido de venda à compra do
-- último componente e vice-versa.
CREATE TABLE IF NOT EXISTS mrp_pegging (
    id             bigserial   PRIMARY KEY,
    enterprise_id  bigint      NOT NULL REFERENCES enterprise(id),
    plan_code      bigint      NOT NULL,
    item_code      bigint      NOT NULL,
    -- Necessidade: PEDIDO_VENDA (sales_order_items.code), DEMANDA_INDEPENDENTE
    -- (independent_demands.code), PREVISAO, MPS, ESTOQUE_SEGURANCA, SUGESTAO
    -- (sugestão do pai), SIMULADA (ajuste de cenário) ou OUTRA.
    demand_kind    varchar(24) NOT NULL,
    demand_code    bigint,
    need_date      date        NOT NULL,
    -- Suprimento: ESTOQUE, ORDEM_FIRME (planned_orders.code), SUGESTAO
    -- (mrp_planned_suggestions.code) ou FALTA (nada cobriu).
    supply_kind    varchar(16) NOT NULL,
    supply_code    bigint,
    supply_date    date,
    quantity       numeric(18,6) NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_mrp_pegging_supply CHECK (supply_kind IN ('ESTOQUE', 'ORDEM_FIRME', 'SUGESTAO', 'FALTA'))
);
CREATE INDEX IF NOT EXISTS idx_mrp_pegging_plan_supply ON mrp_pegging (enterprise_id, plan_code, supply_kind, supply_code);
CREATE INDEX IF NOT EXISTS idx_mrp_pegging_plan_demand ON mrp_pegging (enterprise_id, plan_code, demand_kind, demand_code);

COMMENT ON TABLE mrp_pegging IS
    'Vínculo necessidade × suprimento do último cálculo de cada plano (pegging fixo).';

-- ── Cenários de simulação ──────────────────────────────────────────────────
-- O cenário calcula sobre um plano oculto, clonado do plano base a cada
-- cálculo. Plano oculto não aparece nas listas nem pode ser alterado, firmado
-- ou ter exceção tratada.
ALTER TABLE production_plans ADD COLUMN IF NOT EXISTS is_scenario boolean NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS planning_scenarios (
    id                  bigserial    PRIMARY KEY,
    enterprise_id       bigint       NOT NULL REFERENCES enterprise(id),
    name                varchar(120) NOT NULL,
    description         text,
    base_plan_code      bigint       NOT NULL REFERENCES production_plans(code),
    scenario_plan_code  bigint       NOT NULL UNIQUE REFERENCES production_plans(code),
    -- RASCUNHO (sem cálculo ou ajustes mudaram depois), CALCULADO.
    status              varchar(12)  NOT NULL DEFAULT 'RASCUNHO',
    calculated_at       timestamptz,
    calculation_notes   text,
    promoted_at         timestamptz,
    promoted_by         uuid REFERENCES users(id),
    created_by          uuid NOT NULL REFERENCES users(id),
    created_at          timestamptz  NOT NULL DEFAULT now(),
    updated_at          timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT ck_planning_scenario_status CHECK (status IN ('RASCUNHO', 'CALCULADO'))
);
CREATE INDEX IF NOT EXISTS idx_planning_scenarios_tenant ON planning_scenarios (enterprise_id, base_plan_code);

-- Ajustes do cenário. Um tipo por linha:
--  DEMANDA     — demand_ref_kind NOVA (item, data, quantidade) ou
--                DEMANDA_INDEPENDENTE / PEDIDO_VENDA (demand_ref_code) com nova
--                quantidade e/ou data; quantidade 0 retira a demanda.
--  ESTOQUE     — item, quantity = entrada (+) ou baixa (−) simulada.
--  LEAD_TIME   — item, lead_time_days.
--  CAPACIDADE  — machine_id, hours_per_day e/ou parada (stop_from..stop_to).
CREATE TABLE IF NOT EXISTS planning_scenario_changes (
    id               bigserial   PRIMARY KEY,
    enterprise_id    bigint      NOT NULL REFERENCES enterprise(id),
    scenario_id      bigint      NOT NULL REFERENCES planning_scenarios(id) ON DELETE CASCADE,
    kind             varchar(12) NOT NULL,
    item_code        bigint,
    mask             varchar(200) NOT NULL DEFAULT '',
    demand_ref_kind  varchar(24),
    demand_ref_code  bigint,
    quantity         numeric(18,6),
    need_date        date,
    lead_time_days   integer,
    machine_id       bigint,
    hours_per_day    numeric(6,2),
    stop_from        date,
    stop_to          date,
    notes            text,
    created_by       uuid        NOT NULL REFERENCES users(id),
    created_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_scenario_change_kind CHECK (kind IN ('DEMANDA', 'ESTOQUE', 'LEAD_TIME', 'CAPACIDADE')),
    CONSTRAINT ck_scenario_change_demand CHECK (kind <> 'DEMANDA' OR demand_ref_kind IN ('NOVA', 'DEMANDA_INDEPENDENTE', 'PEDIDO_VENDA')),
    CONSTRAINT ck_scenario_change_hours CHECK (hours_per_day IS NULL OR hours_per_day BETWEEN 0 AND 24),
    CONSTRAINT ck_scenario_change_lead CHECK (lead_time_days IS NULL OR lead_time_days >= 0),
    CONSTRAINT ck_scenario_change_stop CHECK (stop_from IS NULL OR stop_to IS NULL OR stop_to >= stop_from)
);
CREATE INDEX IF NOT EXISTS idx_planning_scenario_changes ON planning_scenario_changes (scenario_id);
