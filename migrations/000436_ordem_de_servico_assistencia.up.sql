-- Pós-venda (Fase 3.3): ordem de serviço da assistência técnica.
--
-- O chamado registrava o problema e podia gerar um pedido ou uma OF, mas não
-- havia onde lançar o atendimento: quem foi, quando, quantas horas, que peças
-- saíram do estoque, quanto se gastou com deslocamento e o que é garantia ou é
-- cobrado do cliente. É a ordem de serviço do SAP CS e do TOTVS Field Service:
-- no cliente (técnico externo) ou na fábrica (o produto volta pela RMA).

-- Parâmetros da OS por empresa.
CREATE TABLE os_parametros (
    enterprise_id         BIGINT        PRIMARY KEY REFERENCES enterprise(id) ON DELETE CASCADE,
    custo_hora            NUMERIC(18,4) NOT NULL DEFAULT 0 CHECK (custo_hora >= 0),
    preco_hora            NUMERIC(18,4) NOT NULL DEFAULT 0 CHECK (preco_hora >= 0),
    custo_km              NUMERIC(18,4) NOT NULL DEFAULT 0 CHECK (custo_km >= 0),
    preco_km              NUMERIC(18,4) NOT NULL DEFAULT 0 CHECK (preco_km >= 0),
    -- Almoxarifado de onde saem as peças da assistência.
    warehouse_id          BIGINT        REFERENCES warehouse(id),
    -- Item de serviço do pedido de faturamento (mão de obra e despesas).
    item_servico_code     BIGINT        REFERENCES items(code),
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_by            UUID          REFERENCES users(id)
);

CREATE TABLE ordens_servico (
    id                   BIGSERIAL     PRIMARY KEY,
    enterprise_id        BIGINT        NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    numero               BIGINT        NOT NULL,
    call_code            BIGINT        NOT NULL REFERENCES technical_assistance_calls(code),
    call_item_code       BIGINT        REFERENCES technical_assistance_call_items(code),
    serie_id             BIGINT,
    item_code            BIGINT        REFERENCES items(code),
    customer_code        BIGINT        NOT NULL,
    local                VARCHAR(8)    NOT NULL CHECK (local IN ('CLIENTE', 'FABRICA')),
    situacao             VARCHAR(12)   NOT NULL DEFAULT 'ABERTA'
                         CHECK (situacao IN ('ABERTA', 'AGENDADA', 'EM_EXECUCAO', 'CONCLUIDA', 'FATURADA', 'CANCELADA')),
    tecnico_code         BIGINT,
    agendada_para        TIMESTAMPTZ,
    iniciada_em          TIMESTAMPTZ,
    concluida_em         TIMESTAMPTZ,
    -- Garantia na abertura (do item do chamado): o que é garantia não é cobrado.
    em_garantia          BOOLEAN       NOT NULL DEFAULT FALSE,
    cobre_pecas          BOOLEAN       NOT NULL DEFAULT FALSE,
    cobre_mao_obra       BOOLEAN       NOT NULL DEFAULT FALSE,
    cobre_deslocamento   BOOLEAN       NOT NULL DEFAULT FALSE,
    defeito_constatado   TEXT,
    solucao              TEXT,
    observacoes          TEXT,
    motivo_cancelamento  TEXT,
    sales_order_code     BIGINT,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by           UUID          REFERENCES users(id),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_by           UUID          REFERENCES users(id),
    UNIQUE (enterprise_id, numero),
    UNIQUE (enterprise_id, id),
    FOREIGN KEY (enterprise_id, serie_id) REFERENCES numeros_serie(enterprise_id, id)
);
CREATE INDEX idx_ordens_servico_chamado ON ordens_servico(enterprise_id, call_code);
CREATE INDEX idx_ordens_servico_situacao ON ordens_servico(enterprise_id, situacao);

-- Apontamento de horas do técnico.
CREATE TABLE os_horas (
    id            BIGSERIAL     PRIMARY KEY,
    enterprise_id BIGINT        NOT NULL,
    os_id         BIGINT        NOT NULL,
    tecnico_code  BIGINT,
    data          DATE          NOT NULL,
    tipo          VARCHAR(12)   NOT NULL CHECK (tipo IN ('TRABALHO', 'DESLOCAMENTO')),
    horas         NUMERIC(10,2) NOT NULL CHECK (horas > 0),
    custo_hora    NUMERIC(18,4) NOT NULL DEFAULT 0,
    preco_hora    NUMERIC(18,4) NOT NULL DEFAULT 0,
    cobrado       BOOLEAN       NOT NULL,
    observacao    TEXT,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by    UUID          REFERENCES users(id),
    FOREIGN KEY (enterprise_id, os_id) REFERENCES ordens_servico(enterprise_id, id) ON DELETE CASCADE
);

-- Peças aplicadas: cada uma é uma saída do estoque, pelo custo médio.
CREATE TABLE os_pecas (
    id                BIGSERIAL     PRIMARY KEY,
    enterprise_id     BIGINT        NOT NULL,
    os_id             BIGINT        NOT NULL,
    item_code         BIGINT        NOT NULL REFERENCES items(code),
    quantidade        NUMERIC(18,6) NOT NULL CHECK (quantidade > 0),
    warehouse_id      BIGINT        NOT NULL REFERENCES warehouse(id),
    custo_unitario    NUMERIC(18,6) NOT NULL DEFAULT 0,
    preco_unitario    NUMERIC(18,6) NOT NULL DEFAULT 0,
    cobrado           BOOLEAN       NOT NULL,
    stock_movement_id BIGINT        REFERENCES stock_movements(id),
    observacao        TEXT,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by        UUID          REFERENCES users(id),
    FOREIGN KEY (enterprise_id, os_id) REFERENCES ordens_servico(enterprise_id, id) ON DELETE CASCADE
);

-- Despesas do atendimento (quilometragem, hospedagem, alimentação, pedágio).
CREATE TABLE os_despesas (
    id            BIGSERIAL     PRIMARY KEY,
    enterprise_id BIGINT        NOT NULL,
    os_id         BIGINT        NOT NULL,
    tipo          VARCHAR(12)   NOT NULL CHECK (tipo IN ('KM', 'HOSPEDAGEM', 'ALIMENTACAO', 'PEDAGIO', 'OUTROS')),
    quantidade    NUMERIC(18,4) NOT NULL DEFAULT 1 CHECK (quantidade > 0),
    custo         NUMERIC(18,4) NOT NULL DEFAULT 0 CHECK (custo >= 0),
    preco         NUMERIC(18,4) NOT NULL DEFAULT 0 CHECK (preco >= 0),
    cobrado       BOOLEAN       NOT NULL,
    observacao    TEXT,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by    UUID          REFERENCES users(id),
    FOREIGN KEY (enterprise_id, os_id) REFERENCES ordens_servico(enterprise_id, id) ON DELETE CASCADE
);

-- Histórico da OS: imutável.
CREATE TABLE os_eventos (
    id            BIGSERIAL   PRIMARY KEY,
    enterprise_id BIGINT      NOT NULL,
    os_id         BIGINT      NOT NULL,
    evento        VARCHAR(20) NOT NULL,
    de            VARCHAR(12),
    para          VARCHAR(12),
    detalhe       TEXT,
    ocorrido_em   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by    UUID        REFERENCES users(id),
    FOREIGN KEY (enterprise_id, os_id) REFERENCES ordens_servico(enterprise_id, id) ON DELETE CASCADE
);
CREATE FUNCTION os_eventos_imutaveis() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'o histórico da ordem de serviço não pode ser alterado';
END $$;
CREATE TRIGGER trg_os_eventos_imutaveis BEFORE UPDATE OR DELETE ON os_eventos
    FOR EACH ROW WHEN (pg_trigger_depth() = 0) EXECUTE FUNCTION os_eventos_imutaveis();

SELECT instalar_conferencia_de_empresa();
