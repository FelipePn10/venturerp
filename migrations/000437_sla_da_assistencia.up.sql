-- Pós-venda (Fase 3.4): SLA da assistência técnica.
--
-- Prazo de PRIMEIRA RESPOSTA e de SOLUÇÃO do chamado, em horas corridas, por
-- prioridade — e, quando houver acordo, por cliente (a regra do cliente vence a
-- geral). O chamado guarda o prazo calculado na abertura (como o SLA do SAP
-- Service e o entitlement da Oracle): mudar a regra não reescreve o prazo de
-- chamado já aberto. A Central de Alertas avisa quando o prazo entra em risco
-- e quando vence.

CREATE TABLE sla_assistencia (
    id            BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT       NOT NULL REFERENCES enterprise(id),
    prioridade    VARCHAR(16)  NOT NULL CHECK (prioridade IN ('LOW', 'NORMAL', 'HIGH', 'CRITICAL')),
    customer_code BIGINT, -- conferido contra a empresa no repositório (customers.code não é único sozinho)
    horas_resposta NUMERIC(8,2) NOT NULL CHECK (horas_resposta > 0),
    horas_solucao  NUMERIC(8,2) NOT NULL CHECK (horas_solucao > 0),
    -- Percentual do prazo consumido a partir do qual o prazo está "em risco".
    alerta_pct    INTEGER      NOT NULL DEFAULT 80 CHECK (alerta_pct BETWEEN 1 AND 99),
    ativo         BOOLEAN      NOT NULL DEFAULT TRUE,
    created_by    UUID         REFERENCES users(id),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by    UUID         REFERENCES users(id),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CHECK (horas_solucao >= horas_resposta),
    UNIQUE NULLS NOT DISTINCT (enterprise_id, prioridade, customer_code)
);

-- Prazos do chamado, gravados na abertura (e na troca de prioridade).
ALTER TABLE technical_assistance_calls
    ADD COLUMN sla_id          BIGINT REFERENCES sla_assistencia(id) ON DELETE SET NULL,
    ADD COLUMN sla_resposta_ate TIMESTAMPTZ,
    ADD COLUMN sla_solucao_ate  TIMESTAMPTZ,
    ADD COLUMN sla_alerta_pct   INTEGER CHECK (sla_alerta_pct BETWEEN 1 AND 99),
    -- Primeira resposta: o chamado saiu de "pendente" ou ganhou uma OS.
    ADD COLUMN respondido_em    TIMESTAMPTZ;
CREATE INDEX idx_ta_calls_sla_aberto ON technical_assistance_calls (enterprise_code, sla_solucao_ate)
    WHERE sla_solucao_ate IS NOT NULL AND status NOT IN ('ATTENDED', 'CLOSED', 'CANCELLED');

-- Alertas do SLA na Central de Alertas (pendências: ficam abertas até o
-- chamado responder/solucionar).
INSERT INTO notification_event_catalog (event_key, version, name_pt_br, description_pt_br, module, event_kind, severity, allowed_cadences,
    template_key, suggested_recipient_roles, producer_status, producer_description)
VALUES
    ('ASSISTENCIA_SLA_EM_RISCO', 1, 'SLA do chamado em risco', 'Chamado consumiu a maior parte do prazo de resposta ou de solução.',
     'ASSISTENCIA', 'PENDENCIA', 'ATENCAO', ARRAY['IMEDIATO', 'RESUMO_DIARIO'], 'alerta_padrao', ARRAY['ASSISTENCIA'],
     'ATIVO', 'Produtor conectado e monitorado pelo backend.'),
    ('ASSISTENCIA_SLA_VENCIDO', 1, 'SLA do chamado vencido', 'Chamado passou do prazo de resposta ou de solução.',
     'ASSISTENCIA', 'PENDENCIA', 'CRITICO', ARRAY['IMEDIATO', 'RESUMO_DIARIO'], 'alerta_padrao', ARRAY['ASSISTENCIA'],
     'ATIVO', 'Produtor conectado e monitorado pelo backend.');

SELECT instalar_conferencia_de_empresa();
