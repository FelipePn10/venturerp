BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Configurador de Produto — núcleo (Fase 1).
--
-- Camada nova e rica, paralela ao antigo `questions`, que modela o configurador
-- no nível do FoccoERP: Conjuntos + Variáveis, Características (com tipos) e
-- Características do Item. A geração/consulta de máscara faz ponte para a tabela
-- `item_masks` já consumida por estrutura/venda/MRP (compatibilidade total).
--
-- Convenção do projeto: VARCHAR + CHECK em vez de ENUM (evita NullXxxEnum no sqlc).
-- ─────────────────────────────────────────────────────────────────────────────

-- ─── Conjuntos (Manutenção de Conjuntos) ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS cfg_sets (
    id          BIGSERIAL PRIMARY KEY,
    description VARCHAR(120) NOT NULL,     -- nome que representa o conjunto
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  UUID NOT NULL
);

-- ─── Variáveis (Manutenção de Variáveis, por conjunto) ────────────────────────
CREATE TABLE IF NOT EXISTS cfg_variables (
    id                  BIGSERIAL PRIMARY KEY,
    set_id              BIGINT NOT NULL REFERENCES cfg_sets(id) ON DELETE CASCADE,
    code                VARCHAR(60) NOT NULL,          -- código da variável
    description         VARCHAR(200) NOT NULL,         -- nome da variável
    mask_composition    VARCHAR(120) NOT NULL,         -- sigla/valor na máscara
    is_active           BOOLEAN NOT NULL DEFAULT TRUE, -- Ativo
    is_special          BOOLEAN NOT NULL DEFAULT FALSE,-- Especial
    include_description  BOOLEAN NOT NULL DEFAULT FALSE,-- Inclui Desc.
    special_data        TEXT,                          -- Dados Esp.
    marketing           BOOLEAN NOT NULL DEFAULT FALSE,-- Mark. (ciclo de vida / marketing)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID NOT NULL,
    UNIQUE (set_id, code)
);
CREATE INDEX IF NOT EXISTS idx_cfg_variables_set ON cfg_variables(set_id);

-- Idiomas da variável (tradução + país)
CREATE TABLE IF NOT EXISTS cfg_variable_languages (
    id          BIGSERIAL PRIMARY KEY,
    variable_id BIGINT NOT NULL REFERENCES cfg_variables(id) ON DELETE CASCADE,
    language    VARCHAR(10) NOT NULL,      -- pt-BR, en-US, es-ES...
    country     VARCHAR(60),
    translation VARCHAR(200) NOT NULL,
    UNIQUE (variable_id, language)
);

-- ─── Características (Manutenção de Características) ────────────────────────────
CREATE TABLE IF NOT EXISTS cfg_characteristics (
    id                  BIGSERIAL PRIMARY KEY,
    code                VARCHAR(60) NOT NULL UNIQUE,   -- código de fácil localização
    description         VARCHAR(200) NOT NULL,         -- pergunta no configurador
    char_type           VARCHAR(20) NOT NULL CHECK (char_type IN (
                            'ESCOLHA','ESCOLHA_MULT','FORMULA','DESENHO','INF_CARACTER',
                            'INF_NUMERICA','OPCAO','CAMPO','SEQUENCIAL')),
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,          -- Ativo/Inativo
    set_id              BIGINT REFERENCES cfg_sets(id),         -- Conjunto (ESCOLHA/ESCOLHA_MULT)
    default_variable_id BIGINT REFERENCES cfg_variables(id),    -- Var. Default
    mask                VARCHAR(120),                           -- Máscara (visualização, # delim.)
    is_special          BOOLEAN NOT NULL DEFAULT FALSE,         -- Especial
    affects_price       BOOLEAN NOT NULL DEFAULT FALSE,         -- Afeta Preço
    controls_goals      BOOLEAN NOT NULL DEFAULT FALSE,         -- Controlar Metas
    receiving_type      VARCHAR(20) NOT NULL DEFAULT 'NENHUM' CHECK (receiving_type IN (
                            'NENHUM','RECEBIMENTO','VINCULO','RECEBIMENTO_VINCULO')),
    -- específicos por tipo
    field_source        VARCHAR(20) CHECK (field_source IN (
                            'ITEM_CODE','CUSTOMER_CODE','ORDER_CODE','SEQUENTIAL')), -- tipo CAMPO
    formula             TEXT,                                   -- tipo FORMULA (default a nível de característica)
    is_required         BOOLEAN NOT NULL DEFAULT FALSE,         -- INF_CARACTER (obrigatório)
    num_min             NUMERIC(15,4),                          -- INF_NUMERICA
    num_max             NUMERIC(15,4),
    num_multiple        NUMERIC(15,4),
    option_true         VARCHAR(60),                            -- OPCAO (rótulo "sim")
    option_false        VARCHAR(60),                            -- OPCAO (rótulo "não")
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID NOT NULL
);

-- Idiomas da característica (descrição/máscara por idioma)
CREATE TABLE IF NOT EXISTS cfg_characteristic_languages (
    id                BIGSERIAL PRIMARY KEY,
    characteristic_id BIGINT NOT NULL REFERENCES cfg_characteristics(id) ON DELETE CASCADE,
    language          VARCHAR(10) NOT NULL,
    description       VARCHAR(200) NOT NULL,
    mask              VARCHAR(120),
    UNIQUE (characteristic_id, language)
);

-- ─── Características do Item (Manutenção de Características do Item) ─────────────
CREATE TABLE IF NOT EXISTS cfg_item_characteristics (
    id                  BIGSERIAL PRIMARY KEY,
    item_code           BIGINT NOT NULL,
    characteristic_id   BIGINT NOT NULL REFERENCES cfg_characteristics(id),
    sequence            INT NOT NULL,                        -- Seq. (10 em 10)
    default_variable_id BIGINT REFERENCES cfg_variables(id), -- Resposta Default (sobrepõe a da característica)
    parent_id           BIGINT REFERENCES cfg_item_characteristics(id) ON DELETE SET NULL, -- Carac. Pai
    is_special          BOOLEAN NOT NULL DEFAULT FALSE,      -- Esp.
    is_drawing          BOOLEAN NOT NULL DEFAULT FALSE,      -- Des.
    is_load             BOOLEAN NOT NULL DEFAULT FALSE,      -- Carga
    formula             TEXT,                                -- fórmula a nível de item (tipo FORMULA)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (item_code, characteristic_id),
    UNIQUE (item_code, sequence)
);
CREATE INDEX IF NOT EXISTS idx_cfg_item_char_item ON cfg_item_characteristics(item_code);

-- Respostas Default (múltiplas — para ESCOLHA_MULT)
CREATE TABLE IF NOT EXISTS cfg_item_char_default_answers (
    id                     BIGSERIAL PRIMARY KEY,
    item_characteristic_id BIGINT NOT NULL REFERENCES cfg_item_characteristics(id) ON DELETE CASCADE,
    variable_id            BIGINT NOT NULL REFERENCES cfg_variables(id),
    UNIQUE (item_characteristic_id, variable_id)
);

-- Respostas ricas que compõem uma máscara gerada (ponte de rastreabilidade sobre
-- item_masks — item_masks.mask continua sendo o que estrutura/venda/MRP consomem).
CREATE TABLE IF NOT EXISTS cfg_item_mask_answers (
    id                BIGSERIAL PRIMARY KEY,
    mask_id           BIGINT NOT NULL REFERENCES item_masks(id) ON DELETE CASCADE,
    characteristic_id BIGINT NOT NULL REFERENCES cfg_characteristics(id),
    variable_id       BIGINT REFERENCES cfg_variables(id),
    answer_value      VARCHAR(200) NOT NULL,   -- valor efetivo na máscara
    position          INT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cfg_item_mask_answers_mask ON cfg_item_mask_answers(mask_id);

COMMIT;
