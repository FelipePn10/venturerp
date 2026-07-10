BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Configurador de Produto — Fase 3: Cadastro de Desenhos + Máscara de Lotes/Séries.
--
-- Desenhos: um cabeçalho (código+dígito+formato) com N revisões. O código de
-- replicação é Desenho(20)+Dígito+Formato+Revisão. Ligado ao item e às
-- características do configurador (tipo DESENHO).
--
-- Máscara de Lotes/Séries: template de geração automática de código de lote,
-- composto por partes ordenadas (Caracter/Data/Sequência Numérica/Sequência
-- Caracter), resolvido por cliente/item/classificação/aplicação.
-- ─────────────────────────────────────────────────────────────────────────────

-- ─── Desenhos ─────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS drawings (
    id            BIGSERIAL PRIMARY KEY,
    code          VARCHAR(60) NOT NULL,        -- Desenho (20 primeiras posições contam p/ replicação)
    digit         VARCHAR(10) NOT NULL DEFAULT '',
    format        VARCHAR(20) NOT NULL DEFAULT '',
    model         VARCHAR(60),
    item_code     BIGINT,                      -- item vinculado (sem FK rígida)
    description   VARCHAR(200),
    uom           VARCHAR(10),
    weight        NUMERIC(15,4),
    material_spec VARCHAR(120),                -- E.M.
    creation_date DATE,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by    UUID NOT NULL,
    UNIQUE (code, digit, format)
);
CREATE INDEX IF NOT EXISTS idx_drawings_item ON drawings(item_code);

-- Revisões do desenho (com vigência, aprovação e motivo)
CREATE TABLE IF NOT EXISTS drawing_revisions (
    id            BIGSERIAL PRIMARY KEY,
    drawing_id    BIGINT NOT NULL REFERENCES drawings(id) ON DELETE CASCADE,
    revision      VARCHAR(20) NOT NULL,
    start_date    DATE,                        -- Data Início da validade
    end_date      DATE,                        -- Data Fim da validade
    material_spec VARCHAR(120),                -- E.M. da revisão
    reason        TEXT,                        -- Motivo da revisão
    approved_by   VARCHAR(120),                -- Aprovação
    approval_date DATE,
    is_current    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (drawing_id, revision)
);
CREATE INDEX IF NOT EXISTS idx_drawing_revisions_drawing ON drawing_revisions(drawing_id);

-- Distribuição de uma revisão (Botão Distribuição)
CREATE TABLE IF NOT EXISTS drawing_revision_distributions (
    id            BIGSERIAL PRIMARY KEY,
    revision_id   BIGINT NOT NULL REFERENCES drawing_revisions(id) ON DELETE CASCADE,
    recipient     VARCHAR(120) NOT NULL,
    distributed_at DATE,
    notes         TEXT
);

-- Vínculo do desenho a características/variáveis do configurador
CREATE TABLE IF NOT EXISTS drawing_characteristics (
    id                BIGSERIAL PRIMARY KEY,
    drawing_id        BIGINT NOT NULL REFERENCES drawings(id) ON DELETE CASCADE,
    characteristic_id BIGINT NOT NULL REFERENCES cfg_characteristics(id),
    operator          VARCHAR(20) NOT NULL DEFAULT 'EQUAL'
                      CHECK (operator IN ('EQUAL','DIFFERENT','GREATER','LESS','BELONGS','NOT_BELONGS')),
    variable_id       BIGINT REFERENCES cfg_variables(id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_drawing_chars_drawing ON drawing_characteristics(drawing_id);

-- ─── Máscara de Lotes/Séries ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS lot_masks (
    id                  BIGSERIAL PRIMARY KEY,
    application         VARCHAR(20) NOT NULL DEFAULT 'GERAL', -- módulo (SUPRIMENTOS/PRODUCAO/VENDAS/EXPEDICAO/GERAL)
    customer_code       BIGINT,                               -- máscara específica p/ cliente
    item_code           BIGINT,                               -- máscara específica p/ item
    classification_type VARCHAR(30),                          -- Clas. Item (tipo)
    classification_code BIGINT,
    zero_on_year_change BOOLEAN NOT NULL DEFAULT FALSE,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    description         VARCHAR(120),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_lot_masks_ctx ON lot_masks(application, customer_code, item_code);

-- Partes da máscara (ordenadas por sequence), com estado da sequência incremental.
CREATE TABLE IF NOT EXISTS lot_mask_parts (
    id            BIGSERIAL PRIMARY KEY,
    lot_mask_id   BIGINT NOT NULL REFERENCES lot_masks(id) ON DELETE CASCADE,
    sequence      INT NOT NULL,                 -- Seq. (ordem na geração)
    part_type     VARCHAR(15) NOT NULL CHECK (part_type IN ('CARACTER','DATA','SEQ_NUMERICA','SEQ_CARACTER')),
    value         VARCHAR(40) NOT NULL DEFAULT '', -- Valor (texto fixo / valor inicial da sequência)
    size          INT NOT NULL DEFAULT 0,          -- Tamanho (0 = automático)
    date_format   VARCHAR(20),                     -- Máscara p/ tipo DATA (ex.: DDMMYYYY)
    zero_on_year_change BOOLEAN NOT NULL DEFAULT FALSE,
    current_value VARCHAR(40) NOT NULL DEFAULT '', -- último valor gerado (estado da sequência)
    last_year     INT,                             -- ano do último incremento (p/ zerar na virada)
    UNIQUE (lot_mask_id, sequence)
);
CREATE INDEX IF NOT EXISTS idx_lot_mask_parts_mask ON lot_mask_parts(lot_mask_id);

COMMIT;
