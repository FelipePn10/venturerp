-- CAPA / 8D: tratamento da não conformidade até a causa raiz e a verificação
-- de eficácia.
--
-- Antes a NC tinha só "causa raiz" e "ação corretiva" em texto livre, sem
-- responsável, prazo, equipe nem verificação — e a NC era fechada pela
-- disposição do material, que é outra coisa (o que fazer com as peças ruins
-- não diz por que elas saíram ruins).
--
-- Método 8D (D0 a D8) ou SIMPLES (análise → ação → verificação) para problema
-- menor. A etapa só avança com o conteúdo dela; o encerramento exige a
-- verificação de eficácia, e ação ineficaz devolve a CAPA à causa raiz.

CREATE TABLE quality_capas (
    id                 BIGSERIAL   PRIMARY KEY,
    enterprise_id      BIGINT      NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    numero             BIGINT      NOT NULL,
    metodo             VARCHAR(8)  NOT NULL CHECK (metodo IN ('8D', 'SIMPLES')),
    tipo               VARCHAR(10) NOT NULL DEFAULT 'CORRETIVA' CHECK (tipo IN ('CORRETIVA', 'PREVENTIVA')),
    origem             VARCHAR(20) NOT NULL CHECK (origem IN ('NC', 'RECLAMACAO_CLIENTE', 'FORNECEDOR', 'AUDITORIA', 'INTERNA')),
    titulo             TEXT        NOT NULL,
    item_code          BIGINT,
    supplier_code      BIGINT,
    customer_code      BIGINT,
    referencia         TEXT,
    responsavel        UUID        NOT NULL REFERENCES users(id),
    prazo              DATE,
    -- 8D: D0 … D8; SIMPLES: ANALISE, ACAO, VERIFICACAO. ENCERRADA/CANCELADA no status.
    etapa              VARCHAR(12) NOT NULL,
    status             VARCHAR(10) NOT NULL DEFAULT 'ABERTA' CHECK (status IN ('ABERTA', 'ENCERRADA', 'CANCELADA')),
    -- D2: descrição do problema (5W2H: o quê, onde, quando, quem, quanto, como, por quê detectou).
    problema           JSONB       NOT NULL DEFAULT '{}',
    -- D3: contenção dispensada precisa de justificativa.
    contencao_dispensada TEXT,
    -- D4: método e análise da causa raiz.
    metodo_causa       VARCHAR(10) CHECK (metodo_causa IN ('5PORQUES', 'ISHIKAWA')),
    cinco_porques      JSONB       NOT NULL DEFAULT '[]',
    ishikawa           JSONB       NOT NULL DEFAULT '{}',
    causa_raiz         TEXT,
    -- Ponto de escape: por que o controle não detectou.
    ponto_de_escape    TEXT,
    -- D7: prevenção de recorrência (plano de inspeção, IT, FMEA, poka-yoke…).
    prevencao          TEXT,
    -- D8 / verificação.
    eficaz             BOOLEAN,
    verificacao        TEXT,
    licoes             TEXT,
    ciclo              INTEGER     NOT NULL DEFAULT 1,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by         UUID        REFERENCES users(id),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at          TIMESTAMPTZ,
    closed_by          UUID        REFERENCES users(id),
    UNIQUE (enterprise_id, numero),
    UNIQUE (enterprise_id, id)
);
CREATE INDEX idx_quality_capas_status ON quality_capas(enterprise_id, status, prazo);

CREATE TABLE quality_capa_ncs (
    capa_id       BIGINT NOT NULL,
    enterprise_id BIGINT NOT NULL,
    nc_id         BIGINT NOT NULL REFERENCES non_conformances(id) ON DELETE CASCADE,
    PRIMARY KEY (capa_id, nc_id),
    FOREIGN KEY (enterprise_id, capa_id) REFERENCES quality_capas(enterprise_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_quality_capa_ncs_nc ON quality_capa_ncs(nc_id);

CREATE TABLE quality_capa_equipe (
    capa_id       BIGINT      NOT NULL,
    enterprise_id BIGINT      NOT NULL,
    user_id       UUID        NOT NULL REFERENCES users(id),
    papel         VARCHAR(10) NOT NULL CHECK (papel IN ('LIDER', 'MEMBRO', 'PATROCINADOR')),
    PRIMARY KEY (capa_id, user_id),
    FOREIGN KEY (enterprise_id, capa_id) REFERENCES quality_capas(enterprise_id, id) ON DELETE CASCADE
);

CREATE TABLE quality_capa_acoes (
    id            BIGSERIAL   PRIMARY KEY,
    capa_id       BIGINT      NOT NULL,
    enterprise_id BIGINT      NOT NULL,
    tipo          VARCHAR(10) NOT NULL CHECK (tipo IN ('CONTENCAO', 'CORRETIVA', 'PREVENTIVA')),
    descricao     TEXT        NOT NULL,
    responsavel   UUID        NOT NULL REFERENCES users(id),
    prazo         DATE        NOT NULL,
    status        VARCHAR(12) NOT NULL DEFAULT 'PENDENTE' CHECK (status IN ('PENDENTE', 'EM_ANDAMENTO', 'CONCLUIDA', 'CANCELADA')),
    evidencia     TEXT,
    concluida_em  TIMESTAMPTZ,
    concluida_por UUID        REFERENCES users(id),
    ciclo         INTEGER     NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by    UUID        REFERENCES users(id),
    FOREIGN KEY (enterprise_id, capa_id) REFERENCES quality_capas(enterprise_id, id) ON DELETE CASCADE,
    CHECK (status <> 'CONCLUIDA' OR (evidencia IS NOT NULL AND concluida_em IS NOT NULL))
);
CREATE INDEX idx_quality_capa_acoes_pendentes ON quality_capa_acoes(enterprise_id, responsavel, prazo) WHERE status IN ('PENDENTE', 'EM_ANDAMENTO');

CREATE TABLE quality_capa_eventos (
    id            BIGSERIAL   PRIMARY KEY,
    capa_id       BIGINT      NOT NULL,
    enterprise_id BIGINT      NOT NULL,
    evento        VARCHAR(20) NOT NULL,
    de            VARCHAR(12),
    para          VARCHAR(12),
    detalhe       TEXT,
    ator          UUID        REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (enterprise_id, capa_id) REFERENCES quality_capas(enterprise_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_quality_capa_eventos ON quality_capa_eventos(capa_id, created_at);

CREATE FUNCTION quality_capa_eventos_imutaveis() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'o histórico da CAPA não pode ser alterado';
END $$;
CREATE TRIGGER trg_quality_capa_eventos_imutaveis BEFORE UPDATE ON quality_capa_eventos
    FOR EACH ROW EXECUTE FUNCTION quality_capa_eventos_imutaveis();

SELECT instalar_conferencia_de_empresa();
