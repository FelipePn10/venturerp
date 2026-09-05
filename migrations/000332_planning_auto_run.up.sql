-- Execução automática do planejamento (MRP → CRP → APS).
--
-- Um MRP que só roda quando alguém clica é um MRP que atrasa. A prática de
-- SAP/TOTVS/Focco é executar em janela noturna, quando ninguém está lançando, e
-- garantir que duas execuções nunca se sobreponham.

-- Parametrização por empresa: se roda sozinho, a que horas e sobre qual plano.
CREATE TABLE IF NOT EXISTS planning_auto_run_settings (
    enterprise_id        bigint      NOT NULL REFERENCES enterprise(id),
    is_enabled           boolean     NOT NULL DEFAULT false,
    -- Hora local da janela noturna. 2h é o padrão: fora do expediente e depois
    -- do fechamento dos apontamentos do dia.
    run_hour             smallint    NOT NULL DEFAULT 2,
    run_minute           smallint    NOT NULL DEFAULT 0,
    -- Plano usado pela execução automática e numeração inicial das ordens.
    plan_code            bigint,
    initial_order_number bigint      NOT NULL DEFAULT 1,
    generate_llc         boolean     NOT NULL DEFAULT true,
    -- Corte da última execução concluída. É o que permite replanejar na próxima
    -- rodada tudo o que mudou durante ou depois da anterior.
    last_snapshot_at     timestamptz,
    last_run_at          timestamptz,
    last_run_status      varchar(20),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pk_planning_auto_run_settings PRIMARY KEY (enterprise_id),
    CONSTRAINT ck_planning_auto_run_hour   CHECK (run_hour BETWEEN 0 AND 23),
    CONSTRAINT ck_planning_auto_run_minute CHECK (run_minute BETWEEN 0 AND 59)
);

COMMENT ON TABLE planning_auto_run_settings IS
    'Janela noturna do planejamento por empresa e corte da última execução.';
COMMENT ON COLUMN planning_auto_run_settings.last_snapshot_at IS
    'Instante lido pela última execução concluída; o que mudou depois entra na próxima.';

-- Histórico de execuções: quem disparou, sobre qual corte e o que saiu.
-- Sem isto não há como explicar por que uma ordem apareceu (ou não) num ciclo.
CREATE TABLE IF NOT EXISTS planning_run_history (
    id             bigserial   PRIMARY KEY,
    enterprise_id  bigint      NOT NULL REFERENCES enterprise(id),
    plan_code      bigint      NOT NULL,
    -- AUTOMATICO (janela noturna) ou MANUAL (usuário clicou).
    trigger        varchar(12) NOT NULL,
    triggered_by   uuid        REFERENCES users(id),
    snapshot_at    timestamptz NOT NULL,
    started_at     timestamptz NOT NULL DEFAULT now(),
    finished_at    timestamptz,
    status         varchar(20) NOT NULL DEFAULT 'RUNNING',
    -- Itens que mudaram desde o corte anterior e por isso foram replanejados.
    changed_items  integer     NOT NULL DEFAULT 0,
    mrp_orders     integer     NOT NULL DEFAULT 0,
    crp_overload   integer     NOT NULL DEFAULT 0,
    viable         boolean,
    notes          text,
    CONSTRAINT ck_planning_run_trigger CHECK (trigger IN ('AUTOMATICO', 'MANUAL')),
    CONSTRAINT ck_planning_run_status  CHECK (status IN ('RUNNING', 'SUCCESS', 'FAILED', 'SKIPPED'))
);

CREATE INDEX IF NOT EXISTS idx_planning_run_history_tenant
    ON planning_run_history (enterprise_id, started_at DESC);

COMMENT ON TABLE planning_run_history IS
    'Uma linha por execução do planejamento, com o corte lido e a origem do disparo.';
