BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Configurador de Produto — Fase 4: Tipos de Descrição + Descrição de Itens
-- Configurados.
--
-- Permite descrever a máscara de um item configurado de formas diferentes por
-- destino (programa/relatório/LOV): por tipo de descrição, define-se para cada
-- característica do item se ela aparece, como (descrição x composição), com que
-- texto e ordem, e se quebra linha.
-- ─────────────────────────────────────────────────────────────────────────────

-- Tipos de Descrição (Manutenção dos Tipos de Descrições)
CREATE TABLE IF NOT EXISTS cfg_description_types (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(30) NOT NULL UNIQUE,   -- Tipo (código)
    description VARCHAR(120) NOT NULL,         -- Descrição
    kind        VARCHAR(20) NOT NULL DEFAULT 'GERAL', -- Tipo da Descrição (PROGRAMA/RELATORIO/LOV/GERAL)
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  UUID NOT NULL
);

-- Descrição de um item configurado para um tipo de descrição (cabeçalho)
CREATE TABLE IF NOT EXISTS cfg_item_descriptions (
    id                  BIGSERIAL PRIMARY KEY,
    item_code           BIGINT NOT NULL,
    description_type_id BIGINT NOT NULL REFERENCES cfg_description_types(id) ON DELETE CASCADE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID NOT NULL,
    UNIQUE (item_code, description_type_id)
);
CREATE INDEX IF NOT EXISTS idx_cfg_item_desc_item ON cfg_item_descriptions(item_code);

-- Linhas da grade (uma por característica do item)
CREATE TABLE IF NOT EXISTS cfg_item_description_lines (
    id                     BIGSERIAL PRIMARY KEY,
    item_description_id    BIGINT NOT NULL REFERENCES cfg_item_descriptions(id) ON DELETE CASCADE,
    item_characteristic_id BIGINT NOT NULL REFERENCES cfg_item_characteristics(id) ON DELETE CASCADE,
    order_index            INT NOT NULL DEFAULT 0,        -- Ord.
    show_characteristic    BOOLEAN NOT NULL DEFAULT TRUE, -- Carac. (Sim/Não)
    show_mask              BOOLEAN NOT NULL DEFAULT TRUE, -- Masc. (Sim/Não)
    desc_type              VARCHAR(15) NOT NULL DEFAULT 'DESCRICAO'
                           CHECK (desc_type IN ('DESCRICAO','COMP_MASCARA')), -- Tipo Desc.
    text                   VARCHAR(120) NOT NULL DEFAULT '', -- Txt
    line_break             BOOLEAN NOT NULL DEFAULT FALSE,   -- Queb. Lin.
    UNIQUE (item_description_id, item_characteristic_id)
);
CREATE INDEX IF NOT EXISTS idx_cfg_item_desc_lines_hdr ON cfg_item_description_lines(item_description_id);

COMMIT;
