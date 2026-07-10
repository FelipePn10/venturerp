BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Configurador de Produto — Fase 5: Regras de Variáveis Equivalentes + Regras de
-- Itens Configurados.
--
-- Regras de Variáveis Equivalentes: mapeiam a configuração do item PAI para a
-- configuração do item FILHO (na estrutura), auto-configurando o filho.
--
-- Regras de Itens Configurados: quando a configuração de um item satisfaz as
-- condições, define o valor de um campo de uma pasta do Cadastro de Item.
-- ─────────────────────────────────────────────────────────────────────────────

-- Regras de Variáveis Equivalentes (pai → filho)
CREATE TABLE IF NOT EXISTS cfg_equivalent_rules (
    id                       BIGSERIAL PRIMARY KEY,
    parent_item_code         BIGINT NOT NULL,
    parent_uom               VARCHAR(10),
    child_item_code          BIGINT NOT NULL,
    child_seq                INT,                          -- sequência do filho na estrutura do pai
    parent_characteristic_id BIGINT NOT NULL REFERENCES cfg_characteristics(id),
    parent_operator          VARCHAR(15) NOT NULL DEFAULT 'EQUAL'
                             CHECK (parent_operator IN ('EQUAL','DIFFERENT','GREATER','LESS','BELONGS','NOT_BELONGS')),
    parent_variable_id       BIGINT REFERENCES cfg_variables(id),
    child_characteristic_id  BIGINT NOT NULL REFERENCES cfg_characteristics(id),
    child_operator           VARCHAR(15) NOT NULL DEFAULT 'EQUAL'
                             CHECK (child_operator IN ('EQUAL','DIFFERENT','GREATER','LESS','BELONGS','NOT_BELONGS')),
    child_variable_id        BIGINT REFERENCES cfg_variables(id),
    formula                  TEXT,                          -- Botão F
    is_active                BOOLEAN NOT NULL DEFAULT TRUE,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by               UUID NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cfg_equiv_rules_parent ON cfg_equivalent_rules(parent_item_code);
CREATE INDEX IF NOT EXISTS idx_cfg_equiv_rules_child ON cfg_equivalent_rules(child_item_code);

-- Regras de Itens Configurados (configuração → campo da pasta do item)
CREATE TABLE IF NOT EXISTS cfg_item_rules (
    id           BIGSERIAL PRIMARY KEY,
    item_code    BIGINT NOT NULL,
    target_table VARCHAR(60) NOT NULL,   -- Tabela (pasta do Cadastro de Item)
    target_field VARCHAR(60) NOT NULL,   -- Campo
    content      VARCHAR(200),           -- Conteúdo (resultado da regra)
    formula      TEXT,                   -- Botão F (função para o resultado)
    description  VARCHAR(200),
    situation    VARCHAR(10) NOT NULL DEFAULT 'ACTIVE' CHECK (situation IN ('ACTIVE','INACTIVE')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by   UUID NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cfg_item_rules_item ON cfg_item_rules(item_code);

-- Condições da regra (bloco Regras): característica OP variável (AND entre linhas)
CREATE TABLE IF NOT EXISTS cfg_item_rule_conditions (
    id                BIGSERIAL PRIMARY KEY,
    rule_id           BIGINT NOT NULL REFERENCES cfg_item_rules(id) ON DELETE CASCADE,
    characteristic_id BIGINT NOT NULL REFERENCES cfg_characteristics(id),
    operator          VARCHAR(15) NOT NULL DEFAULT 'EQUAL'
                      CHECK (operator IN ('EQUAL','DIFFERENT','GREATER','LESS','BELONGS','NOT_BELONGS')),
    variable_id       BIGINT REFERENCES cfg_variables(id),
    sequence          INT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_cfg_item_rule_cond_rule ON cfg_item_rule_conditions(rule_id);

COMMIT;
