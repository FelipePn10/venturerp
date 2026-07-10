BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Configurador — Botão "Itens" do Tipo Recebimento (empresa encarroçadora).
--
-- Para características com Tipo Recebimento RECEBIMENTO / VINCULO / RECEBIMENTO_
-- VINCULO, informa quais respostas (variáveis), itens ou classificações de itens
-- serão usados por tipo de recebimento. Quando nenhuma resposta é informada, o
-- tipo de recebimento da característica vale para todas as respostas.
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS cfg_characteristic_receiving_items (
    id                  BIGSERIAL PRIMARY KEY,
    characteristic_id   BIGINT NOT NULL REFERENCES cfg_characteristics(id) ON DELETE CASCADE,
    variable_id         BIGINT REFERENCES cfg_variables(id),  -- resposta (nulo = toda a característica)
    receiving_type      VARCHAR(20) NOT NULL CHECK (receiving_type IN ('RECEBIMENTO','VINCULO')),
    item_code           BIGINT,                                -- item vinculado (opcional)
    classification_code BIGINT,                                -- classificação de item (opcional)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cfg_char_recv_items_char ON cfg_characteristic_receiving_items(characteristic_id);

COMMIT;
