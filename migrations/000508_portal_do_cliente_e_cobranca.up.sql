-- Portal do Cliente, fase 0 (.ai/tasks/portal-do-cliente.md) e a cobrança
-- bancária sem duplicar (rodada de validação, integrações).

-- Link de acesso do cliente: só o hash SHA-256 do token fica no banco. O cliente
-- é da empresa do link (FK composta), e o token vale até expirar ou ser revogado.
CREATE TABLE IF NOT EXISTS portal_links (
    id              BIGSERIAL PRIMARY KEY,
    enterprise_id   BIGINT      NOT NULL REFERENCES enterprise(id),
    customer_code   BIGINT      NOT NULL,
    token_hash      BYTEA       NOT NULL UNIQUE,
    descricao       TEXT,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    last_access_at  TIMESTAMPTZ,
    access_count    INTEGER     NOT NULL DEFAULT 0,
    created_by      UUID        NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_portal_links_cliente FOREIGN KEY (enterprise_id, customer_code)
        REFERENCES customers (enterprise_id, code)
);
CREATE INDEX IF NOT EXISTS idx_portal_links_cliente ON portal_links (enterprise_id, customer_code);

-- Auditoria do portal: cada acesso e cada pedido feito por ele.
CREATE TABLE IF NOT EXISTS portal_events (
    id             BIGSERIAL PRIMARY KEY,
    enterprise_id  BIGINT      NOT NULL REFERENCES enterprise(id),
    link_id        BIGINT      NOT NULL REFERENCES portal_links(id) ON DELETE CASCADE,
    acao           TEXT        NOT NULL,
    detalhe        TEXT,
    sales_order_code BIGINT,
    -- Na repetição: o pedido de onde vieram os itens (o portal mostra o que o
    -- cliente pediu enquanto o comercial monta o rascunho).
    pedido_origem  BIGINT,
    ip             TEXT,
    user_agent     TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_portal_events_link ON portal_events (link_id, created_at DESC);

-- A cobrança no banco (codigoSolicitacao do Inter): emitir de novo consulta a
-- mesma em vez de criar outra, e cancelar o título cancela a do banco.
ALTER TABLE contas_receber ADD COLUMN IF NOT EXISTS cobranca_externa_id TEXT;

-- Alerta para o comercial quando o cliente pede pelo portal (entrega para quem
-- assinar na Central de Alertas; o cartão da Central de Pendências vale sempre).
INSERT INTO notification_event_catalog (event_key, version, name_pt_br, description_pt_br, module, event_kind, severity,
    allowed_cadences, template_key, enabled_by_default, suggested_recipient_roles, producer_status, producer_description)
VALUES ('COMERCIAL_PEDIDO_PORTAL', 1, 'Pedido feito pelo cliente no portal',
    'O cliente pediu de novo um pedido pelo portal: o rascunho aguarda a confirmação do comercial.',
    'COMERCIAL', 'EVENTO', 'ATENCAO', ARRAY['IMEDIATO','RESUMO_DIARIO','IMEDIATO_E_RESUMO_DIARIO'], 'alerta_padrao', TRUE,
    ARRAY['COMERCIAL'], 'ATIVO', 'Produtor conectado e monitorado pelo backend.')
ON CONFLICT (event_key, version) DO NOTHING;

SELECT instalar_conferencia_de_empresa();
