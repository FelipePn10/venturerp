-- Comercial (Fase 5.2): contrato de venda.
--
-- O acordo de fornecimento com o cliente (quantidade e/ou valor contratados,
-- preço fixado, vigência) era controlado fora do sistema. Agora o contrato tem
-- itens com preço, a programação de entregas, e os pedidos de chamada nascem
-- dele e consomem o saldo — como o contrato de quantidade/valor do SAP e o
-- blanket agreement da Oracle. O vínculo com o pedido fica em tabela própria
-- (o pedido de venda não muda).

CREATE TABLE contratos_venda (
    id              BIGSERIAL     PRIMARY KEY,
    enterprise_id   BIGINT        NOT NULL REFERENCES enterprise(id),
    numero          BIGINT        NOT NULL,
    customer_code   BIGINT        NOT NULL,
    descricao       TEXT          NOT NULL CHECK (btrim(descricao) <> ''),
    vigencia_inicio DATE          NOT NULL,
    vigencia_fim    DATE          NOT NULL,
    -- Teto em valor (opcional): soma das chamadas não passa dele.
    valor_contratado NUMERIC(15,2) CHECK (valor_contratado IS NULL OR valor_contratado > 0),
    situacao        VARCHAR(12)   NOT NULL DEFAULT 'RASCUNHO' CHECK (situacao IN ('RASCUNHO', 'ATIVO', 'ENCERRADO', 'CANCELADO')),
    -- Alerta: saldo abaixo deste percentual do contratado; vencimento em N dias.
    alerta_saldo_pct INTEGER      NOT NULL DEFAULT 20 CHECK (alerta_saldo_pct BETWEEN 0 AND 99),
    alerta_dias      INTEGER      NOT NULL DEFAULT 30 CHECK (alerta_dias BETWEEN 0 AND 365),
    payment_term_code BIGINT,
    observacao      TEXT,
    motivo_encerramento TEXT,
    created_by      UUID          REFERENCES users(id),
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_by      UUID          REFERENCES users(id),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, numero),
    UNIQUE (enterprise_id, id),
    CHECK (vigencia_fim >= vigencia_inicio)
);
CREATE INDEX idx_contratos_venda_cliente ON contratos_venda(enterprise_id, customer_code, situacao);

CREATE TABLE contratos_venda_itens (
    id             BIGSERIAL     PRIMARY KEY,
    enterprise_id  BIGINT        NOT NULL,
    contrato_id    BIGINT        NOT NULL,
    item_code      BIGINT        NOT NULL REFERENCES items(code),
    quantidade     NUMERIC(15,4) NOT NULL CHECK (quantidade > 0),
    preco_unitario NUMERIC(15,6) NOT NULL CHECK (preco_unitario >= 0),
    FOREIGN KEY (enterprise_id, contrato_id) REFERENCES contratos_venda(enterprise_id, id) ON DELETE CASCADE,
    UNIQUE (contrato_id, item_code),
    UNIQUE (enterprise_id, id)
);

-- Programação de entregas: quanto de cada item, para quando.
CREATE TABLE contratos_venda_programacao (
    id              BIGSERIAL     PRIMARY KEY,
    enterprise_id   BIGINT        NOT NULL,
    contrato_item_id BIGINT       NOT NULL,
    data_prevista   DATE          NOT NULL,
    quantidade      NUMERIC(15,4) NOT NULL CHECK (quantidade > 0),
    -- Pedido de chamada que atendeu a parcela (ou nulo).
    sales_order_code BIGINT,
    FOREIGN KEY (enterprise_id, contrato_item_id) REFERENCES contratos_venda_itens(enterprise_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_contratos_venda_programacao ON contratos_venda_programacao(enterprise_id, data_prevista) WHERE sales_order_code IS NULL;

-- Pedidos de chamada: cada linha de pedido que consumiu o contrato.
CREATE TABLE contratos_venda_chamadas (
    id                    BIGSERIAL   PRIMARY KEY,
    enterprise_id         BIGINT      NOT NULL,
    contrato_id           BIGINT      NOT NULL,
    contrato_item_id      BIGINT      NOT NULL,
    sales_order_code      BIGINT      NOT NULL,
    sales_order_item_code BIGINT      NOT NULL REFERENCES sales_order_items(code) ON DELETE CASCADE,
    created_by            UUID        REFERENCES users(id),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (enterprise_id, contrato_id) REFERENCES contratos_venda(enterprise_id, id),
    FOREIGN KEY (enterprise_id, contrato_item_id) REFERENCES contratos_venda_itens(enterprise_id, id),
    UNIQUE (sales_order_item_code)
);
CREATE INDEX idx_contratos_venda_chamadas ON contratos_venda_chamadas(enterprise_id, contrato_id);

INSERT INTO notification_event_catalog (event_key, version, name_pt_br, description_pt_br, module, event_kind, severity, allowed_cadences,
    template_key, suggested_recipient_roles, producer_status, producer_description)
VALUES
    ('COMERCIAL_CONTRATO_VENCENDO', 1, 'Contrato de venda perto do vencimento', 'Contrato de venda ativo vence nos próximos dias.',
     'COMERCIAL', 'PENDENCIA', 'ATENCAO', ARRAY['IMEDIATO', 'RESUMO_DIARIO'], 'alerta_padrao', ARRAY['COMERCIAL'],
     'ATIVO', 'Produtor conectado e monitorado pelo backend.'),
    ('COMERCIAL_CONTRATO_SALDO_BAIXO', 1, 'Contrato de venda com saldo baixo', 'Item do contrato de venda com saldo abaixo do limite de alerta.',
     'COMERCIAL', 'PENDENCIA', 'ATENCAO', ARRAY['IMEDIATO', 'RESUMO_DIARIO'], 'alerta_padrao', ARRAY['COMERCIAL'],
     'ATIVO', 'Produtor conectado e monitorado pelo backend.');

SELECT instalar_conferencia_de_empresa();
