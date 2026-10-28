-- Token Focus NF-e de produção: desde 2026-10-27 só a Venture ERP grava ou
-- troca, pelo comando de servidor `venture-admin`. A tela da empresa mexe
-- apenas na homologação. Cada gesto fica aqui, sem alteração nem exclusão.
CREATE TABLE focus_token_eventos (
    id             BIGSERIAL    PRIMARY KEY,
    enterprise_id  BIGINT       NOT NULL REFERENCES enterprise(id),
    acao           VARCHAR(30)  NOT NULL CHECK (acao IN ('PRODUCAO_GRAVADO', 'VOLTOU_HOMOLOGACAO')),
    token_final    VARCHAR(4),
    executado_por  VARCHAR(120) NOT NULL,
    motivo         TEXT         NOT NULL,
    em             TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_focus_token_eventos_empresa ON focus_token_eventos (enterprise_id, em DESC);

CREATE OR REPLACE FUNCTION focus_token_eventos_imutavel() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'o histórico do token Focus de produção não se altera nem se exclui';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_focus_token_eventos_imutavel
    BEFORE UPDATE OR DELETE ON focus_token_eventos
    FOR EACH ROW EXECUTE FUNCTION focus_token_eventos_imutavel();

SELECT instalar_conferencia_de_empresa();
