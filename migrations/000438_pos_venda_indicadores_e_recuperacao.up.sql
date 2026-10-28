-- Pós-venda (Fase 3.5): ligação com a qualidade e recuperação do custo da
-- garantia junto ao fornecedor.
--
-- O chamado vira reclamação de cliente na qualidade: a CAPA (origem
-- RECLAMACAO_CLIENTE) fica ligada ao chamado. Quando a falha é de componente
-- comprado, o custo da garantia é cobrado do fornecedor (nota de crédito,
-- débito, reposição da peça ou desconto) — como o "supplier recovery" do SAP
-- e da Oracle. Os indicadores (custo de garantia por item/defeito/lote, taxa de
-- falha, MTBF, SLA cumprido) são leitura das tabelas do pós-venda.

ALTER TABLE technical_assistance_calls
    ADD COLUMN capa_id BIGINT REFERENCES quality_capas(id);
CREATE INDEX idx_ta_calls_capa ON technical_assistance_calls(capa_id) WHERE capa_id IS NOT NULL;

CREATE TABLE recuperacoes_fornecedor (
    id               BIGSERIAL     PRIMARY KEY,
    enterprise_id    BIGINT        NOT NULL REFERENCES enterprise(id),
    numero           BIGINT        NOT NULL,
    supplier_code    BIGINT        NOT NULL REFERENCES suppliers(code),
    os_id            BIGINT,
    call_code        BIGINT        REFERENCES technical_assistance_calls(code),
    item_code        BIGINT,
    capa_id          BIGINT,
    motivo           TEXT          NOT NULL CHECK (btrim(motivo) <> ''),
    valor_reclamado  NUMERIC(15,2) NOT NULL CHECK (valor_reclamado > 0),
    valor_recuperado NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (valor_recuperado >= 0 AND valor_recuperado <= valor_reclamado),
    situacao         VARCHAR(12)   NOT NULL DEFAULT 'RECLAMADA' CHECK (situacao IN ('RECLAMADA', 'ACEITA', 'RECUSADA', 'RECUPERADA')),
    forma            VARCHAR(14)   CHECK (forma IN ('NOTA_CREDITO', 'DEBITO', 'REPOSICAO', 'DESCONTO')),
    documento        VARCHAR(80),
    resposta         TEXT,
    recuperado_em    DATE,
    created_by       UUID          REFERENCES users(id),
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_by       UUID          REFERENCES users(id),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, numero),
    FOREIGN KEY (enterprise_id, os_id) REFERENCES ordens_servico(enterprise_id, id),
    FOREIGN KEY (enterprise_id, capa_id) REFERENCES quality_capas(enterprise_id, id),
    CHECK (situacao <> 'RECUPERADA' OR (valor_recuperado > 0 AND forma IS NOT NULL AND recuperado_em IS NOT NULL)),
    CHECK (situacao <> 'RECUSADA' OR resposta IS NOT NULL)
);
CREATE INDEX idx_recuperacoes_fornecedor ON recuperacoes_fornecedor(enterprise_id, situacao, supplier_code);

SELECT instalar_conferencia_de_empresa();
