-- Comercial (Fase 5.3): políticas comerciais e alçada de desconto no pedido.
--
-- As políticas comerciais (desconto, acréscimo, frete e comissão por cliente,
-- segmento, região, condição, tabela, item, faixa de quantidade/valor e
-- vigência) só valiam no orçamento: o pedido digitado direto passava sem elas
-- e o desconto da linha não tinha limite. Agora o pedido aplica as mesmas
-- políticas e o desconto acima da alçada da empresa — ou a política que exige
-- aprovação — bloqueia o pedido até a aprovação (motor de aprovação, tipo
-- PEDIDO_POLITICA_COMERCIAL).

ALTER TABLE aprovacao_politicas DROP CONSTRAINT IF EXISTS aprovacao_politicas_tipo_check;
ALTER TABLE aprovacao_politicas ADD CONSTRAINT aprovacao_politicas_tipo_check
    CHECK (tipo IN ('LIBERACAO_CREDITO', 'ORCAMENTO_POLITICA_COMERCIAL', 'PEDIDO_POLITICA_COMERCIAL'));

CREATE TABLE parametros_comerciais (
    enterprise_id         BIGINT       PRIMARY KEY REFERENCES enterprise(id),
    -- Desconto máximo da linha do pedido sem aprovação (nulo = sem limite).
    desconto_max_pct      NUMERIC(5,2) CHECK (desconto_max_pct IS NULL OR (desconto_max_pct >= 0 AND desconto_max_pct <= 100)),
    updated_by            UUID         REFERENCES users(id),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

SELECT instalar_conferencia_de_empresa();
