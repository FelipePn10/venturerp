BEGIN;

-- ─── Grupo de Estado ───────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS state_groups (
    id          BIGSERIAL PRIMARY KEY,
    code        BIGINT       NOT NULL UNIQUE,
    description VARCHAR(150) NOT NULL,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by  UUID         NOT NULL
);

CREATE TABLE IF NOT EXISTS state_group_ufs (
    id               BIGSERIAL PRIMARY KEY,
    state_group_code BIGINT  NOT NULL REFERENCES state_groups(code) ON DELETE CASCADE,
    uf               CHAR(2) NOT NULL,
    UNIQUE (state_group_code, uf)
);

-- ─── Cadastro de Tipos de Operação de Entrada ──────────────────────────────────
-- nature_operation: natureza da operação (CFOP-like). O 1º dígito determina a regra
-- de validação UF × Grupo de Estado: 1 = dentro do estado, 2 = fora do estado,
-- 3 = fora do país.
CREATE TABLE IF NOT EXISTS entry_operation_types (
    id                  BIGSERIAL PRIMARY KEY,
    code                BIGINT       NOT NULL UNIQUE,
    description         VARCHAR(150) NOT NULL,
    invoice_type_code   BIGINT,                       -- Tipo de Nota
    nature_operation    VARCHAR(10)  NOT NULL,        -- Natureza de Operação
    classification_type VARCHAR(30),                  -- pasta do cadastro de produtos
    classification_code VARCHAR(20),
    state_group_code    BIGINT,                        -- Grupo Estado
    supplier_type_code  BIGINT,                        -- Tipo Fornecedor
    is_active           BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by          UUID         NOT NULL
);

COMMIT;
