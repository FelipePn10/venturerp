-- Perfis de acesso por empresa: permissão por módulo × ação.
--
-- Até aqui o acesso era só o papel do usuário na empresa (ADMIN, USER,
-- OPERATOR, VIEWER), fixo no código. As tabelas perfis_usuario/permissoes da
-- migração 000099 nunca foram usadas e não têm empresa — ficam como estão.
--
-- O perfil é da empresa (cada cliente monta os seus). Um usuário pode ter mais
-- de um perfil na empresa; vale a união. Sem perfil, o usuário segue só com o
-- papel, como antes — a implantação não bloqueia ninguém no dia da atualização.
-- As chaves compostas impedem, no banco, ligar o usuário de uma empresa ao
-- perfil de outra.

CREATE TABLE acesso_perfis (
    id            BIGSERIAL    PRIMARY KEY,
    enterprise_id BIGINT       NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    nome          VARCHAR(80)  NOT NULL CHECK (length(trim(nome)) > 0),
    descricao     TEXT,
    ativo         BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by    UUID         NOT NULL REFERENCES users(id),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by    UUID         NOT NULL REFERENCES users(id),
    UNIQUE (enterprise_id, id)
);
CREATE UNIQUE INDEX uq_acesso_perfis_nome ON acesso_perfis(enterprise_id, lower(nome));

CREATE TABLE acesso_perfil_permissoes (
    perfil_id BIGINT      NOT NULL REFERENCES acesso_perfis(id) ON DELETE CASCADE,
    modulo    VARCHAR(30) NOT NULL CHECK (modulo ~ '^[a-z_]+$'),
    acao      VARCHAR(20) NOT NULL CHECK (acao IN ('LER', 'INCLUIR_ALTERAR', 'EXCLUIR', 'APROVAR', 'AUTORIZAR', 'EXECUTAR')),
    PRIMARY KEY (perfil_id, modulo, acao)
);

CREATE TABLE acesso_usuario_perfis (
    enterprise_id BIGINT      NOT NULL,
    user_id       UUID        NOT NULL,
    perfil_id     BIGINT      NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by    UUID        NOT NULL REFERENCES users(id),
    PRIMARY KEY (enterprise_id, user_id, perfil_id),
    FOREIGN KEY (user_id, enterprise_id) REFERENCES user_enterprises(user_id, enterprise_id) ON DELETE CASCADE,
    FOREIGN KEY (enterprise_id, perfil_id) REFERENCES acesso_perfis(enterprise_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_acesso_usuario_perfis_perfil ON acesso_usuario_perfis(perfil_id);
