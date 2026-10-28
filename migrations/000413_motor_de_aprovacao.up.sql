-- Motor de aprovação genérico: política por empresa e tipo de documento, níveis
-- em escalada, solicitações e eventos imutáveis.
--
-- Primeiros tipos: LIBERACAO_CREDITO (pedido de venda bloqueado pelo limite de
-- crédito) e ORCAMENTO_POLITICA_COMERCIAL (orçamento bloqueado por política
-- comercial que exige aprovação). Antes qualquer USER liberava os dois, sem
-- alçada e sem registro de quem aprovou.
--
-- Níveis em escalada: o valor do documento define até que nível é preciso
-- aprovar (nível 1 até R$ 10 mil; acima, nível 1 e nível 2…). O aprovador do
-- nível é um perfil de acesso da empresa ou um usuário. A segregação de funções
-- (quem pede não aprova; a mesma pessoa não aprova dois níveis) é da política.

CREATE TABLE aprovacao_politicas (
    id            BIGSERIAL   PRIMARY KEY,
    enterprise_id BIGINT      NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    tipo          VARCHAR(40) NOT NULL CHECK (tipo IN ('LIBERACAO_CREDITO', 'ORCAMENTO_POLITICA_COMERCIAL')),
    descricao     TEXT,
    ativa         BOOLEAN     NOT NULL DEFAULT TRUE,
    segregacao    BOOLEAN     NOT NULL DEFAULT TRUE,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by    UUID        NOT NULL REFERENCES users(id),
    UNIQUE (enterprise_id, tipo),
    UNIQUE (enterprise_id, id)
);

CREATE TABLE aprovacao_niveis (
    politica_id   BIGINT        NOT NULL,
    enterprise_id BIGINT        NOT NULL,
    ordem         SMALLINT      NOT NULL CHECK (ordem BETWEEN 1 AND 9),
    -- Valor até o qual este nível encerra a aprovação; NULL = sem teto.
    valor_ate     NUMERIC(18,2) CHECK (valor_ate IS NULL OR valor_ate > 0),
    perfil_id     BIGINT,
    user_id       UUID          REFERENCES users(id),
    PRIMARY KEY (politica_id, ordem),
    FOREIGN KEY (enterprise_id, politica_id) REFERENCES aprovacao_politicas(enterprise_id, id) ON DELETE CASCADE,
    FOREIGN KEY (enterprise_id, perfil_id) REFERENCES acesso_perfis(enterprise_id, id),
    CHECK ((perfil_id IS NULL) <> (user_id IS NULL))
);

CREATE TABLE aprovacao_solicitacoes (
    id               BIGSERIAL     PRIMARY KEY,
    enterprise_id    BIGINT        NOT NULL,
    politica_id      BIGINT        NOT NULL,
    tipo             VARCHAR(40)   NOT NULL,
    documento_codigo BIGINT        NOT NULL,
    valor            NUMERIC(18,2) NOT NULL CHECK (valor >= 0),
    descricao        TEXT          NOT NULL,
    solicitante      UUID          NOT NULL REFERENCES users(id),
    status           VARCHAR(12)   NOT NULL DEFAULT 'PENDENTE' CHECK (status IN ('PENDENTE', 'APROVADA', 'REPROVADA', 'CANCELADA')),
    niveis_exigidos  SMALLINT      NOT NULL CHECK (niveis_exigidos >= 1),
    nivel_atual      SMALLINT      NOT NULL DEFAULT 1,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    concluida_at     TIMESTAMPTZ,
    UNIQUE (enterprise_id, id),
    FOREIGN KEY (enterprise_id, politica_id) REFERENCES aprovacao_politicas(enterprise_id, id)
);
-- Um documento tem no máximo uma solicitação pendente do mesmo tipo.
CREATE UNIQUE INDEX uq_aprovacao_pendente ON aprovacao_solicitacoes(enterprise_id, tipo, documento_codigo) WHERE status = 'PENDENTE';
CREATE INDEX idx_aprovacao_solicitacoes_status ON aprovacao_solicitacoes(enterprise_id, status, created_at DESC);

CREATE TABLE aprovacao_eventos (
    id             BIGSERIAL   PRIMARY KEY,
    enterprise_id  BIGINT      NOT NULL,
    solicitacao_id BIGINT      NOT NULL,
    evento         VARCHAR(20) NOT NULL CHECK (evento IN ('SOLICITADA', 'APROVADA_NIVEL', 'APROVADA', 'REPROVADA', 'CANCELADA')),
    nivel          SMALLINT,
    ator           UUID        NOT NULL REFERENCES users(id),
    justificativa  TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (enterprise_id, solicitacao_id) REFERENCES aprovacao_solicitacoes(enterprise_id, id)
);
CREATE INDEX idx_aprovacao_eventos_solicitacao ON aprovacao_eventos(solicitacao_id, id);

-- Evento é histórico: não se altera nem se apaga.
CREATE FUNCTION aprovacao_eventos_imutaveis() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'eventos de aprovação são imutáveis';
END $$;
CREATE TRIGGER trg_aprovacao_eventos_imutaveis
    BEFORE UPDATE OR DELETE ON aprovacao_eventos
    FOR EACH ROW EXECUTE FUNCTION aprovacao_eventos_imutaveis();
