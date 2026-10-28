-- Pós-venda (SLA, complemento da Fase 3.4): horas úteis e relógio parado.
--
-- 1. A regra pode contar em HORAS ÚTEIS: só o expediente da empresa
--    (sla_expediente; sem cadastro, seg–sex 08–12 e 13–18), menos os feriados
--    do calendário industrial (industrial_calendar.is_workday = false).
-- 2. Enquanto o chamado aguarda o cliente (WAITING_RETURN / WAITING_ORDER), o
--    relógio para; na retomada os prazos em aberto andam o tempo parado. Cada
--    pausa fica registrada em sla_pausas.
-- 3. O momento em que cada prazo entra em risco passa a ser gravado no chamado
--    (com horas úteis ele não é o percentual do intervalo corrido); o alerta
--    da Central usa esse momento.

ALTER TABLE sla_assistencia ADD COLUMN horas_uteis BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE sla_expediente (
    enterprise_id BIGINT      NOT NULL REFERENCES enterprise(id),
    dia_semana    SMALLINT    NOT NULL CHECK (dia_semana BETWEEN 0 AND 6), -- 0 = domingo
    inicio_min    SMALLINT    NOT NULL CHECK (inicio_min BETWEEN 0 AND 1439),
    fim_min       SMALLINT    NOT NULL CHECK (fim_min BETWEEN 1 AND 1440),
    updated_by    UUID        REFERENCES users(id),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_id, dia_semana, inicio_min),
    CHECK (fim_min > inicio_min)
);

ALTER TABLE technical_assistance_calls
    ADD COLUMN sla_horas_uteis      BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN sla_resposta_risco_em TIMESTAMPTZ,
    ADD COLUMN sla_solucao_risco_em  TIMESTAMPTZ,
    ADD COLUMN sla_pausado_desde     TIMESTAMPTZ,
    ADD COLUMN sla_horas_pausadas    NUMERIC(10,2) NOT NULL DEFAULT 0;

-- Chamados já com prazo: o risco é o percentual do intervalo corrido, como era.
UPDATE technical_assistance_calls
   SET sla_resposta_risco_em = opened_at + (sla_resposta_ate - opened_at) * (COALESCE(sla_alerta_pct, 80)::float8 / 100)
 WHERE sla_resposta_ate IS NOT NULL;
UPDATE technical_assistance_calls
   SET sla_solucao_risco_em = opened_at + (sla_solucao_ate - opened_at) * (COALESCE(sla_alerta_pct, 80)::float8 / 100)
 WHERE sla_solucao_ate IS NOT NULL;

CREATE TABLE sla_pausas (
    id            BIGSERIAL   PRIMARY KEY,
    enterprise_id BIGINT      NOT NULL REFERENCES enterprise(id),
    call_code     BIGINT      NOT NULL REFERENCES technical_assistance_calls(code) ON DELETE CASCADE,
    motivo        VARCHAR(20) NOT NULL,
    inicio        TIMESTAMPTZ NOT NULL,
    fim           TIMESTAMPTZ,
    -- Tempo parado no relógio do chamado (útil ou corrido), em horas.
    horas         NUMERIC(10,2),
    CHECK (fim IS NULL OR fim >= inicio)
);
-- Uma pausa aberta por chamado.
CREATE UNIQUE INDEX uq_sla_pausa_aberta ON sla_pausas (call_code) WHERE fim IS NULL;
CREATE INDEX idx_sla_pausas_chamado ON sla_pausas (enterprise_id, call_code);

SELECT instalar_conferencia_de_empresa();
