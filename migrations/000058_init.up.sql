CREATE TYPE situation_enum AS ENUM ('ACTIVE', 'INACTIVE');
CREATE TYPE type_mrp_enum AS ENUM ('MRP', 'MIN_MAX', 'REORDER_POINT', 'MPS', 'KANBAN');
CREATE TYPE type_cc_enum AS ENUM ('AUXILIARY', 'PRODUCTIVE', 'ADMINISTRATIVE', 'COMMERCIAL');
CREATE TYPE type_situation_enum AS ENUM ('ACTIVE', 'INACTIVE', 'PENDING');
CREATE TYPE type_item_enum AS ENUM ('RAW_MATERIAL', 'SEMI_FINISHED', 'FINISHED', 'PACKAGING', 'SERVICE', 'COMPONENT');
CREATE TYPE type_struct_item_enum AS ENUM ('PHANTOM', 'REAL');
CREATE TYPE type_of_use_item_enum AS ENUM ('PURCHASE', 'PRODUCTION', 'OUTSOURCING', 'CONSUMPTION');
CREATE TYPE type_situation_item_enum AS ENUM ('ACTIVE', 'INACTIVE', 'BLOCKED');
CREATE TYPE order_type_enum AS ENUM ('PRODUCTION', 'PURCHASE', 'OUTSOURCING', 'TECHNICAL_ASSISTANCE');
CREATE TYPE order_status_enum AS ENUM ('PLANNED', 'RELEASED', 'IN_PROGRESS', 'FINISHED', 'CANCELLED');
CREATE TYPE sales_division_analysis_enum AS ENUM ('FREE', 'BLOCK_ALWAYS', 'ALWAYS_ANALYZE');
CREATE TYPE restriction_situation_enum AS ENUM ('ACTIVE', 'INACTIVE');
CREATE TYPE restriction_operator_enum AS ENUM ('EQUAL', 'DIFFERENT', 'GREATER', 'LESS', 'BELONGS', 'NOT_BELONGS', 'INVALID');
CREATE TYPE restriction_condition_enum AS ENUM ('AND', 'OR');
CREATE TYPE demand_type_enum AS ENUM ('SALES_ORDER', 'FORECAST', 'INDEPENDENT', 'SAFETY_STOCK', 'REPLENISHMENT');
CREATE TYPE machine_type_enum AS ENUM ('CUT', 'BEND', 'WELD', 'ASSEMBLE', 'PAINT', 'LATHE', 'MILL', 'INJECTION', 'PRESS');

-- 1. INDUSTRIAL CALENDAR
CREATE TABLE industrial_calendar (
                                     id          BIGSERIAL PRIMARY KEY,
                                     year        INT NOT NULL,
                                     month       INT NOT NULL CHECK (month BETWEEN 1 AND 12),
    day         INT NOT NULL CHECK (day BETWEEN 1 AND 31),
    is_workday  BOOLEAN NOT NULL DEFAULT TRUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(year, month, day)
);

CREATE INDEX idx_industrial_calendar_year_month ON industrial_calendar(year, month);
CREATE INDEX idx_industrial_calendar_workday ON industrial_calendar(is_workday);

-- 2. EMPLOYEE / PLANNER
CREATE TABLE employees (
                           id                  BIGSERIAL PRIMARY KEY,
                           code                BIGINT NOT NULL UNIQUE,
                           name                VARCHAR(200) NOT NULL,
                           situation           situation_enum NOT NULL DEFAULT 'ACTIVE',
                           participates_budget BOOLEAN NOT NULL DEFAULT FALSE,
                           technical_assistant BOOLEAN NOT NULL DEFAULT FALSE,
                           role                VARCHAR(50) NOT NULL DEFAULT 'PLANNER', -- PLANNER, OPERATOR, MANAGER, etc
                           created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                           updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                           created_by          UUID NOT NULL
);

CREATE INDEX idx_employees_code ON employees(code);
CREATE INDEX idx_employees_role ON employees(role);

-- 3. COST CENTER

CREATE TABLE cost_centers (
                              id          BIGSERIAL PRIMARY KEY,
                              code        VARCHAR(20) NOT NULL,
                              description VARCHAR(200) NOT NULL,
                              parent_code VARCHAR(20),
                              type        type_cc_enum NOT NULL,
                              is_ratio    BOOLEAN NOT NULL DEFAULT FALSE,
                              start_date  DATE NOT NULL,
                              end_date    DATE,
                              is_active   BOOLEAN NOT NULL DEFAULT TRUE,
                              created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                              updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                              created_by  UUID NOT NULL
);

CREATE INDEX idx_cost_centers_code ON cost_centers(code);
CREATE INDEX idx_cost_centers_type ON cost_centers(type);

-- 4. ALLOCATION BASE / RATEIO

CREATE TABLE allocation_bases (
                                  id          BIGSERIAL PRIMARY KEY,
                                  code        VARCHAR(20) NOT NULL,
                                  description VARCHAR(200) NOT NULL,
                                  period      VARCHAR(7) NOT NULL, -- MM/YYYY
                                  observation TEXT,
                                  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                  created_by  UUID NOT NULL
);

CREATE TABLE allocation_base_items (
                                       id              BIGSERIAL PRIMARY KEY,
                                       allocation_base_id BIGINT NOT NULL REFERENCES allocation_bases(id) ON DELETE CASCADE,
                                       cost_center_id  BIGINT NOT NULL REFERENCES cost_centers(id),
                                       amount          NUMERIC(15,4) NOT NULL DEFAULT 0,
                                       percentage      NUMERIC(5,2) NOT NULL DEFAULT 0,
                                       created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_allocation_bases_code ON allocation_bases(code);
CREATE INDEX idx_allocation_base_items_base ON allocation_base_items(allocation_base_id);

-- 5. OVERHEAD ALLOCATION / RATEIO ABSORCAO
CREATE TABLE overhead_allocations (
                                      id              BIGSERIAL PRIMARY KEY,
                                      cost_center_id  BIGINT NOT NULL REFERENCES cost_centers(id),
                                      plan_account_id BIGINT,
                                      account_code    VARCHAR(20),
                                      period_start    DATE NOT NULL,
                                      period_end      DATE NOT NULL,
                                      allocation_type VARCHAR(30) NOT NULL DEFAULT 'PERCENTAGE', -- BASE, PROPORTIONAL, PERCENTAGE
                                      base_id         BIGINT REFERENCES allocation_bases(id),
                                      created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                      updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                      created_by      UUID NOT NULL
);

CREATE TABLE overhead_allocation_targets (
                                             id                  BIGSERIAL PRIMARY KEY,
                                             overhead_id         BIGINT NOT NULL REFERENCES overhead_allocations(id) ON DELETE CASCADE,
                                             cost_center_id      BIGINT NOT NULL REFERENCES cost_centers(id),
                                             percentage          NUMERIC(5,2) NOT NULL DEFAULT 0,
                                             amount              NUMERIC(15,4) NOT NULL DEFAULT 0,
                                             created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_overhead_allocations_cc ON overhead_allocations(cost_center_id);

-- 6. SALES DIVISION

CREATE TABLE sales_divisions (
                                 id                  BIGSERIAL PRIMARY KEY,
                                 code                BIGINT NOT NULL UNIQUE,
                                 description         VARCHAR(200) NOT NULL,
                                 commercial_analysis sales_division_analysis_enum NOT NULL DEFAULT 'FREE',
                                 financial_analysis  sales_division_analysis_enum NOT NULL DEFAULT 'FREE',
                                 is_technical_assistance BOOLEAN NOT NULL DEFAULT FALSE,
                                 consider_delivery_promise BOOLEAN NOT NULL DEFAULT FALSE,
                                 consider_mrp        BOOLEAN NOT NULL DEFAULT TRUE,
                                 allow_outside_limits BOOLEAN NOT NULL DEFAULT FALSE,
                                 minimum_delivery_days INT NOT NULL DEFAULT 0,
                                 financial_delay_days INT NOT NULL DEFAULT 0,
                                 pis_percentage      NUMERIC(5,2) NOT NULL DEFAULT 0,
                                 cofins_percentage   NUMERIC(5,2) NOT NULL DEFAULT 0,
                                 parent_division_id  BIGINT REFERENCES sales_divisions(id),
                                 is_active           BOOLEAN NOT NULL DEFAULT TRUE,
                                 created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                 updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                 created_by          UUID NOT NULL
);

CREATE INDEX idx_sales_divisions_code ON sales_divisions(code);

-- 7. ORDER PRIORITY
CREATE TABLE order_priorities (
                                  id              BIGSERIAL PRIMARY KEY,
                                  interval_start  NUMERIC(10,2) NOT NULL,
                                  interval_end    NUMERIC(10,2) NOT NULL,
                                  priority        VARCHAR(50) NOT NULL,
                                  description     TEXT,
                                  is_active       BOOLEAN NOT NULL DEFAULT TRUE,
                                  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                  created_by      UUID NOT NULL,
                                  CONSTRAINT chk_interval_valid CHECK (interval_end > interval_start)
);

CREATE INDEX idx_order_priorities_interval ON order_priorities(interval_start, interval_end);

-- 8. RESTRICTIONS / DEPENDENCIES
CREATE TABLE restrictions (
                              id              BIGSERIAL PRIMARY KEY,
                              code            BIGSERIAL,
                              situation       restriction_situation_enum NOT NULL DEFAULT 'ACTIVE',
                              item_code       BIGINT,
                              reason_code     BIGINT,
                              classification_type VARCHAR(50),
                              classification_origin VARCHAR(100),
                              division_id     BIGINT REFERENCES sales_divisions(id),
                              weight          INT NOT NULL DEFAULT 0,
                              created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                              updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                              created_by      UUID NOT NULL
);

CREATE TABLE restriction_dominants (
                                       id              BIGSERIAL PRIMARY KEY,
                                       restriction_id  BIGINT NOT NULL REFERENCES restrictions(id) ON DELETE CASCADE,
                                       question_id     BIGINT NOT NULL,
                                       operator        restriction_operator_enum NOT NULL,
                                       condition_type  restriction_condition_enum NOT NULL DEFAULT 'AND',
                                       answer_value    VARCHAR(100) NOT NULL,
                                       sequence        INT NOT NULL DEFAULT 0
);

CREATE TABLE restriction_determinants (
                                          id              BIGSERIAL PRIMARY KEY,
                                          restriction_id  BIGINT NOT NULL REFERENCES restrictions(id) ON DELETE CASCADE,
                                          question_id     BIGINT NOT NULL,
                                          operator        restriction_operator_enum NOT NULL,
                                          answer_value    VARCHAR(100)
);

CREATE INDEX idx_restrictions_item ON restrictions(item_code);
CREATE INDEX idx_restriction_dominants ON restriction_dominants(restriction_id);

-- 9. PRODUCTION PLAN
CREATE TABLE production_plans (
                                  id                  BIGSERIAL PRIMARY KEY,
                                  code                BIGINT NOT NULL UNIQUE,
                                  name                VARCHAR(200) NOT NULL,
                                  independent_demands VARCHAR(20) NOT NULL DEFAULT 'NO', -- NO, FROM_DATE, ALL
                                  group_same_date_orders BOOLEAN NOT NULL DEFAULT FALSE,
                                  planning_types      TEXT[] NOT NULL DEFAULT ARRAY['MRP'], -- MRP, MIN_MAX, REORDER_POINT, MPS, KANBAN
                                  classification      VARCHAR(50),
                                  class_item_codes    TEXT,
                                  order_item_code     BIGINT,
                                  last_calculated_at  TIMESTAMPTZ,
                                  parameters          JSONB NOT NULL DEFAULT '{}',
                                  is_active           BOOLEAN NOT NULL DEFAULT TRUE,
                                  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                  created_by          UUID NOT NULL
);

CREATE INDEX idx_production_plans_code ON production_plans(code);
CREATE INDEX idx_production_plans_active ON production_plans(is_active);

-- 10. INDEPENDENT DEMANDS
CREATE TABLE independent_demands (
                                     id              BIGSERIAL PRIMARY KEY,
                                     code            BIGINT NOT NULL UNIQUE,
                                     item_code       BIGINT NOT NULL,
                                     mask            VARCHAR(200),
                                     cost_center_id  BIGINT REFERENCES cost_centers(id),
                                     quantity        NUMERIC(15,4) NOT NULL,
                                     demand_date     DATE NOT NULL,
                                     is_active       BOOLEAN NOT NULL DEFAULT TRUE,
                                     created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                     updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                     created_by      UUID NOT NULL
);

CREATE INDEX idx_independent_demands_item ON independent_demands(item_code);
CREATE INDEX idx_independent_demands_date ON independent_demands(demand_date);

-- 11. SALES FORECAST
CREATE TABLE sales_forecasts (
                                 id              BIGSERIAL PRIMARY KEY,
                                 item_code       BIGINT NOT NULL,
                                 mask            VARCHAR(200),
                                 week            INT NOT NULL CHECK (week BETWEEN 1 AND 53),
                                 year            INT NOT NULL,
                                 quantity        NUMERIC(15,4) NOT NULL,
                                 created_by      UUID NOT NULL,
                                 created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                 updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                 UNIQUE(item_code, mask, week, year)
);

CREATE INDEX idx_sales_forecasts_item ON sales_forecasts(item_code);
CREATE INDEX idx_sales_forecasts_period ON sales_forecasts(year, week);

-- 12. SALES FORECAST BLOCK
CREATE TABLE sales_forecast_blocks (
                                       id              BIGSERIAL PRIMARY KEY,
                                       start_date      DATE NOT NULL,
                                       end_date        DATE NOT NULL,
                                       reason          TEXT,
                                       created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                       created_by      UUID NOT NULL
);

-- 13. APPROPRIATION TABLE
CREATE TABLE appropriation_tables (
                                      id          BIGSERIAL PRIMARY KEY,
                                      description VARCHAR(200) NOT NULL,
                                      monday_pct  NUMERIC(5,2) NOT NULL DEFAULT 20,
                                      tuesday_pct NUMERIC(5,2) NOT NULL DEFAULT 20,
                                      wednesday_pct NUMERIC(5,2) NOT NULL DEFAULT 20,
                                      thursday_pct NUMERIC(5,2) NOT NULL DEFAULT 20,
                                      friday_pct  NUMERIC(5,2) NOT NULL DEFAULT 20,
                                      saturday_pct NUMERIC(5,2) NOT NULL DEFAULT 0,
                                      sunday_pct  NUMERIC(5,2) NOT NULL DEFAULT 0,
                                      is_default  BOOLEAN NOT NULL DEFAULT FALSE,
                                      created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                      updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                      created_by  UUID NOT NULL
);

-- 14. DELIVERY PROMISE PARAMETERS
CREATE TABLE delivery_promise_params (
                                         id                          BIGSERIAL PRIMARY KEY,
                                         use_delivery_promise        BOOLEAN NOT NULL DEFAULT FALSE,
                                         blocked_orders_in_promise   BOOLEAN NOT NULL DEFAULT FALSE,
                                         default_order_sort          VARCHAR(20) NOT NULL DEFAULT 'NUM_PEDIDO',
                                         show_order_values           INT NOT NULL DEFAULT 1 CHECK (show_order_values BETWEEN 1 AND 4),
                                         blocked_export_in_promise   BOOLEAN NOT NULL DEFAULT FALSE,
                                         break_tank_occupation       BOOLEAN NOT NULL DEFAULT FALSE,
                                         recalculate_after_release   BOOLEAN NOT NULL DEFAULT FALSE,
                                         reprogram_loaded_orders     BOOLEAN NOT NULL DEFAULT FALSE,
                                         allow_delivery_date_change  BOOLEAN NOT NULL DEFAULT FALSE,
                                         created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                         updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                         updated_by                  UUID NOT NULL
);

-- 15. ITEM CALENDAR PROMISE
CREATE TABLE item_calendar_promises (
                                        id          BIGSERIAL PRIMARY KEY,
                                        item_code   BIGINT NOT NULL,
                                        mask        VARCHAR(200) NOT NULL DEFAULT '',
                                        year        INT NOT NULL,
                                        month       INT NOT NULL CHECK (month BETWEEN 1 AND 12),
    day         INT NOT NULL CHECK (day BETWEEN 1 AND 31),
    is_workday  BOOLEAN NOT NULL DEFAULT TRUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(item_code, mask, year, month, day)
);

CREATE INDEX idx_item_calendar_item ON item_calendar_promises(item_code);

-- 16. DELIVERY RESCHEDULE
CREATE TABLE delivery_reschedules (
                                      id               BIGSERIAL PRIMARY KEY,
                                      code             BIGINT NOT NULL ,
                                      sales_order_code BIGINT NOT NULL,
                                      item_code        BIGINT NOT NULL,
                                      old_date         DATE NOT NULL,
                                      new_date         DATE NOT NULL,
                                      reason           TEXT,
                                      created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                      created_by       UUID NOT NULL
);

CREATE INDEX idx_delivery_reschedule_order ON delivery_reschedules(sales_order_code);

-- 17. PLANNING PARAMETERS
CREATE TABLE planning_params (
                                 id                          BIGSERIAL PRIMARY KEY,
                                 param_number                INT NOT NULL UNIQUE,
                                 param_key                   VARCHAR(50) NOT NULL,
                                 value                       VARCHAR(50) NOT NULL,
                                 description                 TEXT,
                                 created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                 updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                 updated_by                  UUID NOT NULL
);

-- Default parameters
INSERT INTO planning_params (param_number, param_key, value, description, updated_by) VALUES
                                                                                          (1, 'AGRUPA_DEMANDA_ESTOQUE', 'S', 'Agrupa demanda de estoque com primeira demanda do item', '00000000-0000-0000-0000-000000000000'),
                                                                                          (2, 'COD_FORNECEDOR_INTERFABRICA', '', 'Codigo fornecedor para pedido compra inter-fabrica', '00000000-0000-0000-0000-000000000000'),
                                                                                          (3, 'COD_CLIENTE_INTERFABRICA', '', 'Codigo cliente para pedido venda inter-fabrica', '00000000-0000-0000-0000-000000000000'),
                                                                                          (4, 'GERAR_DEMANDA_SEGURANCA_TODOS', 'S', 'Gerar demandas estoque seguranca para todos itens', '00000000-0000-0000-0000-000000000000'),
                                                                                          (5, 'OBRIGATORIEDADE_REFUGO', 'N', 'Obrigatoriedade de refugo na finalizacao de apontamentos', '00000000-0000-0000-0000-000000000000'),
                                                                                          (6, 'DATA_NECESSIDADE_ESTOQUE', 'S', 'Data necessidade das demandas de estoque', '00000000-0000-0000-0000-000000000000'),
                                                                                          (7, 'GERAR_PRIORIDADES_ORDENS', 'S', 'Gerar prioridades nas ordens', '00000000-0000-0000-0000-000000000000'),
                                                                                          (8, 'DIAS_PRIORIDADES', '5', 'Gerar prioridades ate quantos dias', '00000000-0000-0000-0000-000000000000'),
                                                                                          (10, 'ITENS_FANTASMAS_GRAVAR', 'N', 'Itens fantasmas devem ser gravados na ordem de producao', '00000000-0000-0000-0000-000000000000'),
                                                                                          (11, 'DESCONSIDERA_SEMANAS_PASSADAS', 'S', 'Desconsidera semanas ja passadas do mes na previsao', '00000000-0000-0000-0000-000000000000'),
                                                                                          (12, 'CONSIDERA_DATAS_TANQUES', 'N', 'Considerar as datas dos tanques no calculo do MRP', '00000000-0000-0000-0000-000000000000'),
                                                                                          (13, 'VERIFICA_SITUACAO_PEDIDO_PROJETO', 'N', 'Verifica a situacao do pedido de projeto', '00000000-0000-0000-0000-000000000000'),
                                                                                          (14, 'UTILIZA_CALCULO_MPS', 'S', 'Utiliza calculo do MPS', '00000000-0000-0000-0000-000000000000'),
                                                                                          (15, 'PROPORCAO_ENTREGA', 'N', 'Verifica se efetua calculo de proporcionalidade das demandas', '00000000-0000-0000-0000-000000000000'),
                                                                                          (16, 'VALIDA_RESTRICOES_ESTRUTURA', 'S', 'Valida restricoes do item na estrutura', '00000000-0000-0000-0000-000000000000'),
                                                                                          (17, 'TRATA_ASSISTENCIA_TECNICA', 'N', 'Indica se calculo do MRP deve tratar assistencia tecnica', '00000000-0000-0000-0000-000000000000'),
                                                                                          (18, 'PORCENTAGEM_PROPORCAO_VALORIZACAO', '0', '% da proporcao da entrega para relatorio de erros da valorizacao', '00000000-0000-0000-0000-000000000000'),
                                                                                          (19, 'DEFAULT_POSICAO', 'A', 'Default para o campo posicao em alguns programas', '00000000-0000-0000-0000-000000000000'),
                                                                                          (20, 'FORMULA_PERDAS_ESTRUTURA', '2', 'Formula para calculo das perdas na quantidade da estrutura', '00000000-0000-0000-0000-000000000000'),
                                                                                          (24, 'NUMERACAO_ORDENS', 'AUTO', 'Numeracao das ordens geradas', '00000000-0000-0000-0000-000000000000'),
                                                                                          (45, 'OBRIGAR_CONTROLE_ESTOQUE_TERCEIROS', 'N', 'Obrigar controle de estoque de terceiros e em terceiros', '00000000-0000-0000-0000-000000000000');

-- 18. MACHINE TYPE
CREATE TABLE machine_types (
                               id              BIGSERIAL PRIMARY KEY,
                               code            BIGINT NOT NULL UNIQUE,
                               name            VARCHAR(100) NOT NULL,
                               description     TEXT,
                               type            machine_type_enum NOT NULL,
                               setup_time      NUMERIC(10,2) NOT NULL DEFAULT 0, -- tempo de preparo em minutos
                               is_active       BOOLEAN NOT NULL DEFAULT TRUE,
                               created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                               updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                               created_by      UUID NOT NULL
);

-- 19. MACHINE
CREATE TABLE machines (
                          id              BIGSERIAL PRIMARY KEY,
                          code            BIGINT NOT NULL UNIQUE,
                          name            VARCHAR(100) NOT NULL,
                          machine_type_id BIGINT NOT NULL REFERENCES machine_types(id),
                          cost_center_id  BIGINT REFERENCES cost_centers(id),
                          capacity_per_hour NUMERIC(15,4) NOT NULL DEFAULT 0,
                          efficiency_rate NUMERIC(5,2) NOT NULL DEFAULT 100,
                          is_active       BOOLEAN NOT NULL DEFAULT TRUE,
                          created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                          updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                          created_by      UUID NOT NULL
);

CREATE INDEX idx_machines_type ON machines(machine_type_id);
CREATE INDEX idx_machines_code ON machines(code);

-- 20. ITEM-MACHINE ASSOCIATION (production time per item per machine)
CREATE TABLE item_machine_times (
                                    id                  BIGSERIAL PRIMARY KEY,
                                    item_code           BIGINT NOT NULL,
                                    mask                VARCHAR(200) NOT NULL DEFAULT '',
                                    machine_id          BIGINT NOT NULL REFERENCES machines(id),
                                    production_time     NUMERIC(15,4) NOT NULL, -- tempo em minutos por unidade
                                    setup_time          NUMERIC(10,2) NOT NULL DEFAULT 0,
                                    priority            INT NOT NULL DEFAULT 0, -- prioridade na maquina
                                    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
                                    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                    UNIQUE(item_code, mask, machine_id)
);

CREATE INDEX idx_item_machine_times_item ON item_machine_times(item_code);
CREATE INDEX idx_item_machine_times_machine ON item_machine_times(machine_id);

-- 21. PLANNED ORDERS (generated by MRP)
CREATE TABLE planned_orders (
                                id              BIGSERIAL PRIMARY KEY,
                                order_number    BIGINT NOT NULL,
                                item_code       BIGINT NOT NULL,
                                mask            VARCHAR(200) NOT NULL DEFAULT '',
                                quantity        NUMERIC(15,4) NOT NULL,
                                quantity_loss   NUMERIC(15,4) NOT NULL DEFAULT 0,
                                quantity_corrected NUMERIC(15,4) NOT NULL DEFAULT 0,
                                order_type      order_type_enum NOT NULL,
                                status          order_status_enum NOT NULL DEFAULT 'PLANNED',
                                plan_id         BIGINT REFERENCES production_plans(id),
                                demand_type     demand_type_enum NOT NULL,
                                demand_id       BIGINT,
                                need_date       DATE NOT NULL,
                                start_date      DATE,
                                end_date        DATE,
                                cost_center_id  BIGINT REFERENCES cost_centers(id),
                                employee_id     BIGINT REFERENCES employees(id),
                                machine_id      BIGINT REFERENCES machines(id),
                                production_time NUMERIC(15,4) DEFAULT 0,
                                priority        VARCHAR(50),
                                llc             INT NOT NULL DEFAULT 0,
                                notes           TEXT,
                                parent_order_id BIGINT REFERENCES planned_orders(id),
                                sales_order_id  BIGINT,
                                is_firm         BOOLEAN NOT NULL DEFAULT FALSE,
                                is_active       BOOLEAN NOT NULL DEFAULT TRUE,
                                created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                created_by      UUID NOT NULL
);

CREATE INDEX idx_planned_orders_order_number ON planned_orders(order_number);
CREATE INDEX idx_planned_orders_item ON planned_orders(item_code);
CREATE INDEX idx_planned_orders_plan ON planned_orders(plan_id);
CREATE INDEX idx_planned_orders_dates ON planned_orders(need_date);
CREATE INDEX idx_planned_orders_type ON planned_orders(order_type);
CREATE INDEX idx_planned_orders_status ON planned_orders(status);

-- 22. MRP CALCULATION RESULT
CREATE TABLE mrp_item_profiles (
                                   id              BIGSERIAL PRIMARY KEY,
                                   item_code       BIGINT NOT NULL,
                                   plan_id         BIGINT NOT NULL REFERENCES production_plans(id),
                                   calculation_date DATE NOT NULL,
                                   demand          NUMERIC(15,4) NOT NULL DEFAULT 0,
                                   orders_planned  NUMERIC(15,4) NOT NULL DEFAULT 0,
                                   orders_firm     NUMERIC(15,4) NOT NULL DEFAULT 0,
                                   stock_projected NUMERIC(15,4) NOT NULL DEFAULT 0,
                                   llc             INT NOT NULL DEFAULT 0,
                                   need_date       DATE NOT NULL,
                                   created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mrp_profiles_item ON mrp_item_profiles(item_code);
CREATE INDEX idx_mrp_profiles_plan ON mrp_item_profiles(plan_id);

-- 23. MRP CALCULATION LOG
CREATE TABLE mrp_calculation_logs (
                                      id              BIGSERIAL PRIMARY KEY,
                                      plan_id         BIGINT NOT NULL REFERENCES production_plans(id),
                                      started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                      finished_at     TIMESTAMPTZ,
                                      status          VARCHAR(20) NOT NULL DEFAULT 'RUNNING', -- RUNNING, COMPLETED, ERROR
                                      errors          JSONB,
                                      total_items     INT NOT NULL DEFAULT 0,
                                      total_orders    INT NOT NULL DEFAULT 0,
                                      created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mrp_logs_plan ON mrp_calculation_logs(plan_id);

-- 24. MACHINE SCHEDULE (daily production queue per machine)
CREATE TABLE machine_schedules (
                                   id              BIGSERIAL PRIMARY KEY,
                                   machine_id      BIGINT NOT NULL REFERENCES machines(id),
                                   order_id        BIGINT NOT NULL REFERENCES planned_orders(id),
                                   schedule_date   DATE NOT NULL,
                                   start_time      TIME,
                                   end_time        TIME,
                                   planned_qty     NUMERIC(15,4) NOT NULL,
                                   produced_qty    NUMERIC(15,4) NOT NULL DEFAULT 0,
                                   status          VARCHAR(20) NOT NULL DEFAULT 'SCHEDULED', -- SCHEDULED, IN_PROGRESS, COMPLETED, CANCELLED
                                   sequence        INT NOT NULL DEFAULT 0,
                                   priority_override INT DEFAULT NULL, -- when user manually reorders
                                   notes           TEXT,
                                   created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                   updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_machine_schedules_machine ON machine_schedules(machine_id);
CREATE INDEX idx_machine_schedules_date ON machine_schedules(schedule_date);

-- 25. STOCK SNAPSHOT (for MRP calculation)
CREATE TABLE stock_snapshots (
                                 id              BIGSERIAL PRIMARY KEY,
                                 item_code       BIGINT NOT NULL,
                                 warehouse_id    BIGINT NOT NULL,
                                 quantity        NUMERIC(15,4) NOT NULL DEFAULT 0,
                                 reserved_qty    NUMERIC(15,4) NOT NULL DEFAULT 0,
                                 safety_stock    NUMERIC(15,4) NOT NULL DEFAULT 0,
                                 snapshot_date   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                 created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stock_snapshots_item ON stock_snapshots(item_code);

-- 26. SALES ORDER DEMAND (for MRP)
CREATE TABLE sales_order_demands (
                                     id              BIGSERIAL PRIMARY KEY,
                                     sales_order_code  BIGINT NOT NULL,
                                     item_code       BIGINT NOT NULL,
                                     mask            VARCHAR(200) NOT NULL DEFAULT '',
                                     quantity        NUMERIC(15,4) NOT NULL,
                                     delivered_qty   NUMERIC(15,4) NOT NULL DEFAULT 0,
                                     delivery_date   DATE NOT NULL,
                                     division_id     BIGINT REFERENCES sales_divisions(id),
                                     status          VARCHAR(20) NOT NULL DEFAULT 'PENDING',
                                     is_active       BOOLEAN NOT NULL DEFAULT TRUE,
                                     created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                     updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sales_order_demands_item ON sales_order_demands(item_code);
CREATE INDEX idx_sales_order_demands_date ON sales_order_demands(delivery_date);

-- 27. CONFIGURED ITEM RULES
CREATE TABLE configured_item_rules (
                                       id              BIGSERIAL PRIMARY KEY,
                                       item_code       BIGINT NOT NULL,
                                       table_type      VARCHAR(50) NOT NULL, -- PLANNING_DATA, PLANNER_DATA
                                       field_name      VARCHAR(50) NOT NULL,
                                       rule_type       VARCHAR(20) NOT NULL, -- EQUAL, DIFFERENT, RANGE
                                       rule_value      TEXT NOT NULL,
                                       sequence        INT NOT NULL DEFAULT 0,
                                       is_active       BOOLEAN NOT NULL DEFAULT TRUE,
                                       created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                       updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                       created_by      UUID NOT NULL
);

CREATE INDEX idx_configured_rules_item ON configured_item_rules(item_code);
