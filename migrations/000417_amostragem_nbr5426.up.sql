-- Amostragem por atributos da NBR 5426 (= ISO 2859-1).
--
-- O procedimento diz COMO amostrar: pela norma (nível + NQA, com troca de
-- severidade), por plano fixo (n/Ac/Re da empresa) ou 100 %. O estado guarda
-- a severidade corrente de cada unidade de controle — item × fornecedor no
-- recebimento, item × operação no processo — porque a troca normal ↔ apertada
-- ↔ atenuada depende do histórico daquele fornecedor naquele item, não do item
-- sozinho. O evento registra cada troca e cada suspensão (não se altera).

CREATE TABLE quality_sampling_procedures (
    id               BIGSERIAL   PRIMARY KEY,
    enterprise_id    BIGINT      NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    codigo           VARCHAR(20) NOT NULL,
    descricao        TEXT        NOT NULL,
    tipo             VARCHAR(10) NOT NULL CHECK (tipo IN ('NORMA', 'FIXO', 'CEM')),
    nivel            VARCHAR(3)  CHECK (nivel IN ('S1', 'S2', 'S3', 'S4', 'I', 'II', 'III')),
    nqa              VARCHAR(6),
    -- "Autoridade responsável" de 9.3.3 c): sem ela não há inspeção atenuada.
    permite_atenuada BOOLEAN     NOT NULL DEFAULT FALSE,
    amostra_fixa     INTEGER     CHECK (amostra_fixa > 0),
    ac_fixo          INTEGER     CHECK (ac_fixo >= 0),
    re_fixo          INTEGER,
    -- Skip-lot como regra da empresa (não é a ISO 2859-3).
    skip_apos        INTEGER     CHECK (skip_apos > 0),
    skip_frequencia  INTEGER     CHECK (skip_frequencia > 1),
    ativo            BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by       UUID        NOT NULL REFERENCES users(id),
    UNIQUE (enterprise_id, codigo),
    UNIQUE (enterprise_id, id),
    CHECK (tipo <> 'NORMA' OR (nivel IS NOT NULL AND nqa IS NOT NULL)),
    CHECK (tipo <> 'FIXO' OR (amostra_fixa IS NOT NULL AND ac_fixo IS NOT NULL AND re_fixo > ac_fixo)),
    CHECK ((skip_apos IS NULL) = (skip_frequencia IS NULL))
);

CREATE TABLE quality_sampling_states (
    enterprise_id        BIGINT      NOT NULL,
    procedure_id         BIGINT      NOT NULL,
    item_code            BIGINT      NOT NULL,
    -- 'F:<fornecedor>', 'O:<operação do roteiro>' ou '-' (sem unidade).
    chave                VARCHAR(40) NOT NULL,
    severidade           VARCHAR(8)  NOT NULL DEFAULT 'NORMAL' CHECK (severidade IN ('NORMAL', 'APERTADA', 'ATENUADA')),
    pontuacao            INTEGER     NOT NULL DEFAULT 0,
    ultimos              BOOLEAN[]   NOT NULL DEFAULT '{}',
    recusas_em_apertada  INTEGER     NOT NULL DEFAULT 0,
    suspensa             BOOLEAN     NOT NULL DEFAULT FALSE,
    aceitos_seguidos     INTEGER     NOT NULL DEFAULT 0,
    lotes_desde_inspecao INTEGER     NOT NULL DEFAULT 0,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_id, procedure_id, item_code, chave),
    FOREIGN KEY (enterprise_id, procedure_id) REFERENCES quality_sampling_procedures(enterprise_id, id) ON DELETE CASCADE
);

CREATE TABLE quality_sampling_events (
    id            BIGSERIAL   PRIMARY KEY,
    enterprise_id BIGINT      NOT NULL,
    procedure_id  BIGINT      NOT NULL,
    item_code     BIGINT      NOT NULL,
    chave         VARCHAR(40) NOT NULL,
    de            VARCHAR(8)  NOT NULL,
    para          VARCHAR(8)  NOT NULL,
    -- SUSPENSAO, LIBERACAO ou TROCA.
    evento        VARCHAR(10) NOT NULL CHECK (evento IN ('TROCA', 'SUSPENSAO', 'LIBERACAO')),
    motivo        TEXT        NOT NULL,
    referencia    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by    UUID        REFERENCES users(id),
    FOREIGN KEY (enterprise_id, procedure_id) REFERENCES quality_sampling_procedures(enterprise_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_quality_sampling_events_item ON quality_sampling_events(enterprise_id, item_code, created_at DESC);

CREATE FUNCTION quality_sampling_events_imutaveis() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'o histórico de severidade da amostragem não pode ser alterado';
END $$;
CREATE TRIGGER trg_quality_sampling_events_imutaveis BEFORE UPDATE ON quality_sampling_events
    FOR EACH ROW EXECUTE FUNCTION quality_sampling_events_imutaveis();
