-- Pós-venda (Fase 3.1): número de série e base instalada.
--
-- A série existia só como texto solto no movimento de estoque e no item do
-- chamado: nada dizia que a série X saiu na nota Y para o cliente Z, e o
-- chamado não tinha como achar o equipamento. É o que o SAP chama de
-- equipamento/número de série e a TOTVS de Base de Atendimento: o vínculo
-- equipamento × cliente pela série, de onde partem garantia e assistência.

-- Perfil de pós-venda por item e empresa. A 3.2 acrescenta a garantia aqui.
CREATE TABLE item_pos_venda (
    enterprise_id     BIGINT      NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    item_code         BIGINT      NOT NULL REFERENCES items(code),
    controla_serie    BOOLEAN     NOT NULL DEFAULT FALSE,
    -- Máscara de lote/série (VLOT0100) usada para gerar a série na entrega da OF.
    mascara_serie_id  BIGINT      REFERENCES lot_masks(id),
    observacoes       TEXT,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by        UUID        REFERENCES users(id),
    PRIMARY KEY (enterprise_id, item_code)
);

CREATE TABLE numeros_serie (
    id                 BIGSERIAL   PRIMARY KEY,
    enterprise_id      BIGINT      NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    item_code          BIGINT      NOT NULL REFERENCES items(code),
    mask               TEXT,
    numero_serie       VARCHAR(80) NOT NULL CHECK (btrim(numero_serie) <> ''),
    situacao           VARCHAR(16) NOT NULL CHECK (situacao IN ('EM_ESTOQUE', 'NO_CLIENTE', 'EM_ASSISTENCIA', 'BAIXADO')),
    origem             VARCHAR(10) NOT NULL CHECK (origem IN ('PRODUCAO', 'COMPRA', 'CADASTRO')),
    warehouse_id       BIGINT      REFERENCES warehouse(id),
    lote               TEXT,
    production_order_id BIGINT     REFERENCES production_orders(id),
    fiscal_entry_id    BIGINT      REFERENCES fiscal_entries(id),
    -- Venda: a NF de saída autorizada que levou a série ao cliente.
    fiscal_exit_id     BIGINT      REFERENCES fiscal_exits(id),
    customer_code      BIGINT,
    data_venda         DATE,
    data_instalacao    DATE,
    local_instalacao   TEXT,
    observacoes        TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by         UUID        REFERENCES users(id),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by         UUID        REFERENCES users(id),
    UNIQUE (enterprise_id, item_code, numero_serie),
    UNIQUE (enterprise_id, id),
    -- Série no cliente tem cliente; em estoque tem almoxarifado.
    CHECK (situacao <> 'NO_CLIENTE' OR customer_code IS NOT NULL),
    CHECK (situacao <> 'EM_ESTOQUE' OR warehouse_id IS NOT NULL)
);
CREATE INDEX idx_numeros_serie_cliente ON numeros_serie(enterprise_id, customer_code) WHERE customer_code IS NOT NULL;
CREATE INDEX idx_numeros_serie_estoque ON numeros_serie(enterprise_id, item_code, warehouse_id) WHERE situacao = 'EM_ESTOQUE';
CREATE INDEX idx_numeros_serie_busca ON numeros_serie(enterprise_id, upper(numero_serie));

-- Histórico da série: imutável.
CREATE TABLE numeros_serie_eventos (
    id               BIGSERIAL   PRIMARY KEY,
    enterprise_id    BIGINT      NOT NULL,
    serie_id         BIGINT      NOT NULL,
    evento           VARCHAR(20) NOT NULL,
    situacao_antes   VARCHAR(16),
    situacao_depois  VARCHAR(16) NOT NULL,
    customer_code    BIGINT,
    warehouse_id     BIGINT      REFERENCES warehouse(id),
    referencia_tipo  VARCHAR(30),
    referencia_id    BIGINT,
    motivo           TEXT,
    ocorrido_em      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by       UUID        REFERENCES users(id),
    FOREIGN KEY (enterprise_id, serie_id) REFERENCES numeros_serie(enterprise_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_numeros_serie_eventos_serie ON numeros_serie_eventos(serie_id, ocorrido_em);

CREATE FUNCTION numeros_serie_eventos_imutaveis() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'o histórico da série não pode ser alterado';
END $$;
CREATE TRIGGER trg_numeros_serie_eventos_imutaveis BEFORE UPDATE OR DELETE ON numeros_serie_eventos
    FOR EACH ROW WHEN (pg_trigger_depth() = 0) EXECUTE FUNCTION numeros_serie_eventos_imutaveis();

-- Séries escolhidas na linha da NF de saída. Uma série só fica numa nota viva
-- por vez; a nota cancelada (ou a linha refeita) solta a série.
CREATE TABLE fiscal_exit_item_series (
    id                  BIGSERIAL   PRIMARY KEY,
    enterprise_id       BIGINT      NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    fiscal_exit_id      BIGINT      NOT NULL REFERENCES fiscal_exits(id) ON DELETE CASCADE,
    fiscal_exit_item_id BIGINT      NOT NULL REFERENCES fiscal_exit_items(id) ON DELETE CASCADE,
    serie_id            BIGINT      NOT NULL,
    ativo               BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID        REFERENCES users(id),
    FOREIGN KEY (enterprise_id, serie_id) REFERENCES numeros_serie(enterprise_id, id)
);
CREATE UNIQUE INDEX uq_fiscal_exit_item_series_viva ON fiscal_exit_item_series(serie_id) WHERE ativo;
CREATE INDEX idx_fiscal_exit_item_series_nota ON fiscal_exit_item_series(fiscal_exit_id) WHERE ativo;

SELECT instalar_conferencia_de_empresa();
