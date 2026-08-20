ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE notification_event_catalog (
    event_key VARCHAR(120) NOT NULL,
    version INTEGER NOT NULL CHECK (version > 0),
    name_pt_br TEXT NOT NULL,
    description_pt_br TEXT NOT NULL,
    module VARCHAR(40) NOT NULL,
	event_kind VARCHAR(12) NOT NULL DEFAULT 'EVENTO' CHECK(event_kind IN('EVENTO','PENDENCIA')),
    severity VARCHAR(12) NOT NULL CHECK (severity IN ('INFORMATIVO','ATENCAO','CRITICO')),
    allowed_cadences TEXT[] NOT NULL,
    template_key VARCHAR(120) NOT NULL,
    enabled_by_default BOOLEAN NOT NULL DEFAULT FALSE,
    suggested_recipient_roles TEXT[] NOT NULL DEFAULT '{}',
    minimum_payload_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    deduplication_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    resolution_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_key, version),
    CHECK (allowed_cadences <@ ARRAY['IMEDIATO','RESUMO_DIARIO','IMEDIATO_E_RESUMO_DIARIO']::TEXT[])
);

CREATE TABLE enterprise_notification_settings (
    enterprise_id BIGINT PRIMARY KEY REFERENCES enterprise(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    digest_time TIME NOT NULL DEFAULT '08:00',
    timezone VARCHAR(100) NOT NULL DEFAULT 'America/Sao_Paulo',
    retention_days INTEGER NOT NULL DEFAULT 365 CHECK (retention_days BETWEEN 30 AND 3650),
    max_attachment_bytes BIGINT NOT NULL DEFAULT 10485760 CHECK (max_attachment_bytes BETWEEN 0 AND 26214400),
	max_emails_per_minute INTEGER NOT NULL DEFAULT 60 CHECK (max_emails_per_minute BETWEEN 1 AND 1000),
    fiscal_config_id BIGINT UNIQUE REFERENCES fiscal_configs(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE enterprise_departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    code VARCHAR(60) NOT NULL,
    name VARCHAR(120) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(enterprise_id,code),
    UNIQUE(enterprise_id,id)
);

CREATE TABLE enterprise_department_users (
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    department_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(enterprise_id,department_id,user_id),
    FOREIGN KEY(enterprise_id,department_id) REFERENCES enterprise_departments(enterprise_id,id) ON DELETE CASCADE,
    FOREIGN KEY(user_id,enterprise_id) REFERENCES user_enterprises(user_id,enterprise_id) ON DELETE CASCADE
);

CREATE TABLE enterprise_notification_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    event_key VARCHAR(120) NOT NULL,
    event_version INTEGER NOT NULL DEFAULT 1,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    cadence VARCHAR(30) NOT NULL CHECK (cadence IN ('IMEDIATO','RESUMO_DIARIO','IMEDIATO_E_RESUMO_DIARIO')),
    thresholds JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, event_key),
    FOREIGN KEY (event_key, event_version) REFERENCES notification_event_catalog(event_key, version)
);

CREATE TABLE enterprise_notification_recipients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES enterprise_notification_subscriptions(id) ON DELETE CASCADE,
    recipient_type VARCHAR(20) NOT NULL CHECK (recipient_type IN ('USUARIO','PAPEL','DEPARTAMENTO')),
    user_id UUID REFERENCES users(id),
    recipient_key VARCHAR(120),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((recipient_type='USUARIO' AND user_id IS NOT NULL AND recipient_key IS NULL) OR
           (recipient_type<>'USUARIO' AND user_id IS NULL AND recipient_key IS NOT NULL)),
    UNIQUE NULLS NOT DISTINCT (enterprise_id, subscription_id, recipient_type, user_id, recipient_key)
);

CREATE TABLE notification_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE RESTRICT,
    event_key VARCHAR(120) NOT NULL,
    event_version INTEGER NOT NULL DEFAULT 1,
    aggregate_type VARCHAR(80) NOT NULL,
    aggregate_internal_id TEXT,
    aggregate_public_id TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    deduplication_key VARCHAR(240) NOT NULL,
    correlation_id UUID NOT NULL DEFAULT gen_random_uuid(),
    originator_user_id UUID REFERENCES users(id),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    state VARCHAR(20) NOT NULL DEFAULT 'PENDENTE' CHECK (state IN ('PENDENTE','PROCESSANDO','ENVIADO','FALHOU','DESCARTADO','CANCELADO')),
    lease_owner VARCHAR(120),
    lease_until TIMESTAMPTZ,
    last_error_code VARCHAR(80),
    last_error_message VARCHAR(500),
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, event_key, deduplication_key),
    FOREIGN KEY (event_key, event_version) REFERENCES notification_event_catalog(event_key, version)
);
CREATE INDEX idx_notification_outbox_claim ON notification_outbox (next_attempt_at, created_at)
    WHERE state IN ('PENDENTE','FALHOU','PROCESSANDO');
CREATE INDEX idx_notification_outbox_tenant ON notification_outbox (enterprise_id, created_at DESC);

CREATE TABLE notification_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE RESTRICT,
    event_key VARCHAR(120) NOT NULL,
    event_version INTEGER NOT NULL DEFAULT 1,
    aggregate_type VARCHAR(80) NOT NULL,
    aggregate_internal_id TEXT,
    aggregate_public_id TEXT,
    cycle INTEGER NOT NULL DEFAULT 1,
    deduplication_key VARCHAR(240) NOT NULL,
    severity VARCHAR(12) NOT NULL CHECK (severity IN ('INFORMATIVO','ATENCAO','CRITICO')),
    state VARCHAR(12) NOT NULL DEFAULT 'ABERTO' CHECK (state IN ('ABERTO','RESOLVIDO','IGNORADO')),
    summary TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolution_reason VARCHAR(240),
    UNIQUE (enterprise_id, event_key, deduplication_key, cycle),
    FOREIGN KEY (event_key, event_version) REFERENCES notification_event_catalog(event_key, version)
);
CREATE UNIQUE INDEX uq_notification_alert_open ON notification_alerts (enterprise_id, event_key, deduplication_key)
    WHERE state='ABERTO';
CREATE INDEX idx_notification_alert_tenant_state ON notification_alerts (enterprise_id, state, opened_at DESC);

CREATE TABLE notification_digest_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE RESTRICT,
    local_date DATE NOT NULL,
    timezone VARCHAR(100) NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'PENDENTE' CHECK (state IN ('PENDENTE','PROCESSANDO','ENVIADO','FALHOU','DESCARTADO','CANCELADO')),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, local_date)
);

CREATE TABLE notification_digest_items (
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE RESTRICT,
    digest_run_id UUID NOT NULL REFERENCES notification_digest_runs(id) ON DELETE CASCADE,
    recipient_user_id UUID NOT NULL REFERENCES users(id),
    alert_id UUID NOT NULL REFERENCES notification_alerts(id) ON DELETE RESTRICT,
    module_snapshot VARCHAR(40) NOT NULL,
    severity_snapshot VARCHAR(12) NOT NULL CHECK(severity_snapshot IN('INFORMATIVO','ATENCAO','CRITICO')),
    summary_snapshot TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(digest_run_id,recipient_user_id,alert_id)
);
CREATE INDEX idx_notification_digest_items_tenant ON notification_digest_items(enterprise_id,digest_run_id,recipient_user_id);

CREATE TABLE notification_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE RESTRICT,
    outbox_id UUID REFERENCES notification_outbox(id) ON DELETE SET NULL,
    digest_run_id UUID REFERENCES notification_digest_runs(id) ON DELETE SET NULL,
    channel VARCHAR(30) NOT NULL DEFAULT 'EMAIL' CHECK (channel IN ('EMAIL','NOTIFICACAO_INTERNA','WEBHOOK')),
    recipient_user_id UUID NOT NULL REFERENCES users(id),
    recipient_email_snapshot TEXT NOT NULL,
    recipient_name_snapshot TEXT NOT NULL,
    subject_snapshot VARCHAR(240) NOT NULL,
    message_id VARCHAR(255) NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'PENDENTE' CHECK (state IN ('PENDENTE','PROCESSANDO','ENVIADO','FALHOU','DESCARTADO','CANCELADO')),
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lease_owner VARCHAR(120),
    lease_until TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    last_error_code VARCHAR(80),
    last_error_message VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, message_id)
);
CREATE INDEX idx_notification_delivery_claim ON notification_deliveries (next_attempt_at, created_at)
    WHERE state IN ('PENDENTE','FALHOU','PROCESSANDO');
CREATE INDEX idx_notification_delivery_tenant ON notification_deliveries (enterprise_id, created_at DESC);

CREATE TABLE notification_delivery_attempts (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE RESTRICT,
    delivery_id UUID NOT NULL REFERENCES notification_deliveries(id) ON DELETE RESTRICT,
    attempt_number INTEGER NOT NULL CHECK (attempt_number > 0),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    outcome VARCHAR(20) NOT NULL CHECK (outcome IN ('PROCESSANDO','ENVIADO','FALHOU','DESCARTADO')),
    provider_code VARCHAR(80),
    sanitized_error VARCHAR(500),
    manual_retry_by UUID REFERENCES users(id),
    UNIQUE (delivery_id, attempt_number)
);

CREATE TABLE notification_delivery_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE RESTRICT,
    delivery_id UUID NOT NULL REFERENCES notification_deliveries(id) ON DELETE CASCADE,
    file_name VARCHAR(160) NOT NULL CHECK(file_name !~ E'[\\r\\n]'),
    mime_type VARCHAR(80) NOT NULL CHECK(mime_type IN ('application/pdf','text/csv','image/png','image/jpeg')),
    content BYTEA NOT NULL,
    size_bytes BIGINT GENERATED ALWAYS AS (octet_length(content)) STORED,
    sha256 CHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK(octet_length(content)<=26214400),
    UNIQUE(delivery_id,sha256)
);
CREATE INDEX idx_notification_attachment_tenant ON notification_delivery_attachments(enterprise_id,delivery_id);

CREATE TABLE notification_provider_rate_windows (
    scope_key VARCHAR(80) NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    sent_count INTEGER NOT NULL DEFAULT 0 CHECK(sent_count>=0),
    PRIMARY KEY(scope_key,window_start)
);

CREATE TABLE notification_provider_circuit (
    provider_key VARCHAR(60) PRIMARY KEY,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    state VARCHAR(12) NOT NULL DEFAULT 'FECHADO' CHECK(state IN('FECHADO','ABERTO','SEMIABERTO')),
    opened_until TIMESTAMPTZ,
    probe_owner VARCHAR(120),
    probe_lease_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO notification_provider_circuit(provider_key) VALUES('EMAIL_CENTRAL');

CREATE TABLE notification_dead_letters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE RESTRICT,
    delivery_id UUID NOT NULL REFERENCES notification_deliveries(id) ON DELETE RESTRICT,
    reason_code VARCHAR(80) NOT NULL,
    sanitized_reason VARCHAR(500) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    retried_at TIMESTAMPTZ,
    retried_by UUID REFERENCES users(id),
    UNIQUE (delivery_id)
);
CREATE INDEX idx_notification_dead_letter_tenant ON notification_dead_letters (enterprise_id, created_at DESC);

CREATE TABLE stock_cycle_counts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE RESTRICT,
    warehouse_id BIGINT NOT NULL,
    warehouse_address_id BIGINT,
    item_code BIGINT NOT NULL,
    mask VARCHAR(200) NOT NULL DEFAULT '',
    lot_code VARCHAR(120) NOT NULL DEFAULT '',
    scheduled_for TIMESTAMPTZ NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'PROGRAMADA' CHECK (state IN ('PROGRAMADA','EM_CONTAGEM','DIVERGENTE','CONCLUIDA','APROVADA','CANCELADA')),
    expected_quantity NUMERIC(20,6),
    counted_quantity NUMERIC(20,6),
    divergence_quantity NUMERIC(20,6),
    counted_by UUID REFERENCES users(id),
    approved_by UUID REFERENCES users(id),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, warehouse_id, warehouse_address_id, item_code, mask, lot_code, scheduled_for)
);
CREATE INDEX idx_stock_cycle_count_due ON stock_cycle_counts (enterprise_id, state, scheduled_for);

CREATE TABLE stock_cycle_count_audit (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE RESTRICT,
    cycle_count_id UUID NOT NULL REFERENCES stock_cycle_counts(id) ON DELETE RESTRICT,
    action VARCHAR(30) NOT NULL,
    previous_state VARCHAR(20),
    new_state VARCHAR(20) NOT NULL,
    actor_user_id UUID REFERENCES users(id),
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_stock_cycle_count_audit_tenant ON stock_cycle_count_audit (enterprise_id, cycle_count_id, created_at);

WITH events(event_key,name_pt_br,description_pt_br,module,severity,cadences,template_key,roles) AS (VALUES
('COMERCIAL_ORCAMENTO_CONVERTIDO_PEDIDO','Orçamento convertido em pedido','Conversão concluída de orçamento em pedido de venda.','COMERCIAL','INFORMATIVO',ARRAY['IMEDIATO','RESUMO_DIARIO','IMEDIATO_E_RESUMO_DIARIO'],'comercial_orcamento_convertido',ARRAY['COMERCIAL']),
('COMERCIAL_PEDIDO_BLOQUEADO_CREDITO','Pedido bloqueado por crédito','Pedido bloqueado pela análise de crédito.','COMERCIAL','ATENCAO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['COMERCIAL']),
('COMERCIAL_PEDIDO_SEM_PREVISAO_ENTREGA','Pedido sem previsão de entrega','Pedido aberto sem previsão de entrega.','COMERCIAL','ATENCAO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['COMERCIAL']),
('COMERCIAL_PEDIDO_ATRASADO','Pedido atrasado','Pedido com entrega vencida.','COMERCIAL','CRITICO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['COMERCIAL']),
('COMERCIAL_PEDIDO_CANCELADO','Pedido cancelado','Pedido de venda cancelado.','COMERCIAL','ATENCAO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['COMERCIAL']),
('COMERCIAL_MARGEM_ABAIXO_LIMITE','Margem abaixo do limite','Margem comercial abaixo do limite configurado.','COMERCIAL','ATENCAO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['COMERCIAL']),
('COMERCIAL_ORCAMENTO_PROXIMO_VENCIMENTO','Orçamento próximo do vencimento','Orçamento próximo da validade.','COMERCIAL','ATENCAO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['COMERCIAL']),
('COMERCIAL_PEDIDO_ALTERADO_APOS_LIBERACAO','Pedido alterado após liberação','Pedido liberado sofreu alteração relevante.','COMERCIAL','CRITICO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['COMERCIAL']),
('ESTOQUE_CONTAGEM_PROXIMA_VENCIMENTO','Contagem próxima do vencimento','Contagem cíclica próxima do prazo.','ESTOQUE','ATENCAO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['ESTOQUE']),
('ESTOQUE_CONTAGEM_VENCIDA','Contagem vencida','Contagem cíclica não realizada no prazo.','ESTOQUE','CRITICO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['ESTOQUE']),
('ESTOQUE_CONTAGEM_DIVERGENCIA','Divergência de contagem','Contagem cíclica com divergência.','ESTOQUE','CRITICO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['ESTOQUE']),
('ESTOQUE_CONTAGEM_CONCLUIDA','Contagem concluída','Contagem cíclica concluída.','ESTOQUE','INFORMATIVO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['ESTOQUE']),
('ESTOQUE_CONTAGEM_APROVADA','Contagem aprovada','Contagem cíclica aprovada.','ESTOQUE','INFORMATIVO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['ESTOQUE']),
('ESTOQUE_ABAIXO_MINIMO','Estoque abaixo do mínimo','Saldo abaixo do estoque mínimo.','ESTOQUE','ATENCAO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['ESTOQUE']),
('ESTOQUE_NEGATIVO','Estoque negativo','Saldo de estoque negativo.','ESTOQUE','CRITICO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['ESTOQUE']),
('ESTOQUE_LOTE_PROXIMO_VENCIMENTO','Lote próximo do vencimento','Lote próximo da data de validade.','ESTOQUE','ATENCAO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['ESTOQUE']),
('ESTOQUE_MOVIMENTACAO_INCOMUM','Movimentação incomum','Movimentação excedeu regra explícita configurada.','ESTOQUE','ATENCAO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['ESTOQUE']),
('FISCAL_NFE_SAIDA_CRIADA','NF-e de saída criada','NF-e de saída criada.','FISCAL','INFORMATIVO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['FISCAL']),
('FISCAL_NFE_SAIDA_AGUARDANDO_AUTORIZACAO','NF-e aguardando autorização','NF-e de saída aguardando autorização.','FISCAL','ATENCAO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['FISCAL']),
('FISCAL_NFE_SAIDA_AUTORIZADA','NF-e de saída autorizada','NF-e de saída autorizada.','FISCAL','INFORMATIVO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['FISCAL']),
('FISCAL_NFE_SAIDA_REJEITADA','NF-e de saída rejeitada','NF-e de saída rejeitada.','FISCAL','CRITICO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['FISCAL']),
('FISCAL_NFE_SAIDA_CANCELADA','NF-e de saída cancelada','NF-e de saída cancelada.','FISCAL','ATENCAO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['FISCAL']),
('FISCAL_NFE_ENTRADA_IMPORTADA','NF-e de entrada importada','NF-e de entrada importada.','FISCAL','INFORMATIVO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['FISCAL']),
('FISCAL_NFE_ENTRADA_APROVADA','NF-e de entrada aprovada','NF-e de entrada aprovada.','FISCAL','INFORMATIVO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['FISCAL']),
('FISCAL_NFE_ENTRADA_ITEM_NAO_IDENTIFICADO','Item de NF-e não identificado','NF-e de entrada contém item não identificado.','FISCAL','CRITICO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['FISCAL']),
('FISCAL_NFE_ENTRADA_DIVERGENCIA_FISCAL','Divergência fiscal na entrada','NF-e de entrada contém divergência fiscal.','FISCAL','CRITICO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['FISCAL']),
('FISCAL_NFE_ENTRADA_DIVERGENCIA_QUANTIDADE','Divergência de quantidade na entrada','NF-e de entrada contém divergência de quantidade.','FISCAL','ATENCAO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['FISCAL']),
('FISCAL_NFE_ENTRADA_CANCELADA','NF-e de entrada cancelada','NF-e de entrada cancelada.','FISCAL','ATENCAO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['FISCAL']),
('CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS','Item configurado sem perguntas','Item configurado sem característica ativa.','CADASTRO','CRITICO',ARRAY['IMEDIATO_E_RESUMO_DIARIO'],'cadastro_item_sem_perguntas',ARRAY['CADASTRO']),
('CADASTRO_ITEM_CONFIGURADO_INCOMPLETO','Item configurado incompleto','Cadastro configurável incompleto.','CADASTRO','ATENCAO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['CADASTRO']),
('CADASTRO_ITEM_SEM_CLASSIFICACAO_FISCAL','Item sem classificação fiscal','Item sem classificação fiscal.','CADASTRO','CRITICO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['CADASTRO']),
('CADASTRO_ITEM_SEM_UNIDADE_BASE','Item sem unidade base','Item sem unidade de medida base.','CADASTRO','CRITICO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['CADASTRO']),
('CADASTRO_ITEM_SEM_PARAMETROS_MRP','Item sem parâmetros MRP','Item sem parâmetros obrigatórios do MRP.','CADASTRO','ATENCAO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['CADASTRO']),
('CADASTRO_ITEM_COMPRADO_SEM_FORNECEDOR','Item comprado sem fornecedor','Item comprado sem fornecedor válido.','CADASTRO','ATENCAO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['COMPRAS']),
('CADASTRO_ITEM_FABRICADO_SEM_ESTRUTURA','Item fabricado sem estrutura','Item fabricado sem estrutura vigente.','CADASTRO','CRITICO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['PRODUCAO']),
('CADASTRO_ITEM_FABRICADO_SEM_ROTEIRO','Item fabricado sem roteiro','Item fabricado sem roteiro vigente.','CADASTRO','CRITICO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['PRODUCAO']),
('CADASTRO_ITEM_COM_CODIGO_FORNECEDOR_DUPLICADO','Código de fornecedor duplicado','Código de item do fornecedor duplicado.','CADASTRO','ATENCAO',ARRAY['RESUMO_DIARIO'],'alerta_padrao',ARRAY['COMPRAS']),
('CADASTRO_ITEM_INATIVO_COM_DEMANDA_ABERTA','Item inativo com demanda','Item inativo possui demanda aberta.','CADASTRO','CRITICO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY['PLANEJAMENTO'])
)
INSERT INTO notification_event_catalog(event_key,version,name_pt_br,description_pt_br,module,severity,allowed_cadences,template_key,suggested_recipient_roles)
SELECT event_key,1,name_pt_br,description_pt_br,module,severity,cadences,template_key,roles FROM events;

-- Eventos adicionais da matriz são catalogados com template seguro genérico.
WITH keys(event_key,module,name_pt_br) AS (VALUES
('COMPRAS_REQUISICAO_AGUARDANDO_APROVACAO','COMPRAS','Requisição aguardando aprovação'),('COMPRAS_COTACAO_SEM_RESPOSTA','COMPRAS','Cotação sem resposta próxima do prazo'),('COMPRAS_PEDIDO_ATRASADO','COMPRAS','Pedido de compra atrasado'),('COMPRAS_ITEM_SEM_FORNECEDOR_VALIDO','COMPRAS','Item comprado sem fornecedor válido'),('COMPRAS_PRECO_VENCIDO_VARIACAO','COMPRAS','Preço vencido ou fora do limite'),('COMPRAS_LAUDO_AUSENTE_RECEBIMENTO','COMPRAS','Laudo obrigatório ausente'),('COMPRAS_DOCUMENTO_FORNECEDOR_VENCIMENTO','COMPRAS','Documento de fornecedor próximo do vencimento'),('COMPRAS_RECEBIMENTO_DIVERGENCIA','COMPRAS','Divergência no recebimento'),('COMPRAS_LEAD_TIME_ACIMA_CONTRATADO','COMPRAS','Lead time acima do contratado'),
('MRP_EXCECAO','MRP','Exceção do MRP'),('PRODUCAO_OF_ATRASADA_PARADA','PRODUCAO','Ordem de fabricação atrasada ou parada'),('PRODUCAO_OPERACAO_SEM_APONTAMENTO','PRODUCAO','Operação sem apontamento'),('PRODUCAO_CONSUMO_REFUGO_ACIMA_TOLERANCIA','PRODUCAO','Consumo ou refugo acima da tolerância'),('PRODUCAO_FALTA_MATERIAL_OF_LIBERADA','PRODUCAO','Falta de material para OF liberada'),('APS_CAPACIDADE_SOBRECARREGADA','APS','Capacidade sobrecarregada'),('APS_PROMESSA_ENTREGA_EM_RISCO','APS','Promessa de entrega em risco'),('MANUTENCAO_PREVENTIVA_VENCIDA_PROXIMA','MANUTENCAO','Manutenção preventiva vencida ou próxima'),('FERRAMENTA_VIDA_UTIL_PROXIMA_EXCEDIDA','PRODUCAO','Ferramenta próxima ou acima da vida útil'),('CALENDARIO_INDUSTRIAL_INCOMPLETO','PLANEJAMENTO','Calendário industrial incompleto'),
('QUALIDADE_INSPECAO_RECEBIMENTO_PENDENTE','QUALIDADE','Inspeção de recebimento pendente'),('QUALIDADE_NAO_CONFORMIDADE_CRITICA','QUALIDADE','Não conformidade crítica aberta ou vencida'),('QUALIDADE_LAUDO_CERTIFICADO_AUSENTE','QUALIDADE','Laudo ou certificado ausente'),('QUALIDADE_PLANO_OBRIGATORIO_NAO_EXECUTADO','QUALIDADE','Plano obrigatório não executado'),('QUALIDADE_LOTE_BLOQUEADO_REPROVADO','QUALIDADE','Lote bloqueado ou reprovado'),('QUALIDADE_REINCIDENCIA_ACIMA_LIMITE','QUALIDADE','Reincidência acima do limite'),
('FINANCEIRO_TITULO_PROXIMO_VENCIMENTO','FINANCEIRO','Título próximo do vencimento'),('FINANCEIRO_TITULO_VENCIDO','FINANCEIRO','Título vencido'),('FINANCEIRO_BLOQUEIO_LIMITE_CREDITO','FINANCEIRO','Bloqueio ou limite de crédito'),('FINANCEIRO_CONCILIACAO_PENDENTE','FINANCEIRO','Conciliação pendente'),('FINANCEIRO_REMESSA_RETORNO_REJEITADO','FINANCEIRO','Remessa ou retorno bancário rejeitado'),('FINANCEIRO_FLUXO_CAIXA_ABAIXO_LIMITE','FINANCEIRO','Fluxo de caixa abaixo do limite'),
('OPERACAO_INTEGRACAO_FISCAL_FALHA','OPERACAO','Integração fiscal com falha persistente'),('OPERACAO_BACKLOG_DEAD_LETTERS_ANORMAL','OPERACAO','Backlog ou cartas mortas anormais'),('OPERACAO_JOB_ESSENCIAL_SEM_EXECUCAO','OPERACAO','Job essencial sem execução'),('OPERACAO_IMPORTACAO_REJEITADA_REPETIDAMENTE','OPERACAO','Importação rejeitada repetidamente'),('SEGURANCA_TENTATIVA_CROSS_TENANT','SEGURANCA','Tentativa entre empresas bloqueada'),('OPERACAO_ATUALIZACAO_BACKUP_FALHOU','OPERACAO','Atualização ou backup com falha')
)
INSERT INTO notification_event_catalog(event_key,version,name_pt_br,description_pt_br,module,severity,allowed_cadences,template_key,suggested_recipient_roles)
SELECT event_key,1,name_pt_br,name_pt_br||'.',module,'ATENCAO',ARRAY['IMEDIATO','RESUMO_DIARIO'],'alerta_padrao',ARRAY[module] FROM keys;

UPDATE notification_event_catalog SET event_kind='PENDENCIA' WHERE event_key IN(
'COMERCIAL_PEDIDO_BLOQUEADO_CREDITO','COMERCIAL_PEDIDO_SEM_PREVISAO_ENTREGA','COMERCIAL_PEDIDO_ATRASADO','COMERCIAL_MARGEM_ABAIXO_LIMITE','COMERCIAL_ORCAMENTO_PROXIMO_VENCIMENTO',
'ESTOQUE_CONTAGEM_PROXIMA_VENCIMENTO','ESTOQUE_CONTAGEM_VENCIDA','ESTOQUE_CONTAGEM_DIVERGENCIA','ESTOQUE_ABAIXO_MINIMO','ESTOQUE_NEGATIVO','ESTOQUE_LOTE_PROXIMO_VENCIMENTO',
'FISCAL_NFE_SAIDA_AGUARDANDO_AUTORIZACAO','FISCAL_NFE_SAIDA_REJEITADA','FISCAL_NFE_ENTRADA_ITEM_NAO_IDENTIFICADO','FISCAL_NFE_ENTRADA_DIVERGENCIA_FISCAL','FISCAL_NFE_ENTRADA_DIVERGENCIA_QUANTIDADE',
'CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS','CADASTRO_ITEM_CONFIGURADO_INCOMPLETO','CADASTRO_ITEM_SEM_CLASSIFICACAO_FISCAL','CADASTRO_ITEM_SEM_UNIDADE_BASE','CADASTRO_ITEM_SEM_PARAMETROS_MRP','CADASTRO_ITEM_COMPRADO_SEM_FORNECEDOR','CADASTRO_ITEM_FABRICADO_SEM_ESTRUTURA','CADASTRO_ITEM_FABRICADO_SEM_ROTEIRO','CADASTRO_ITEM_COM_CODIGO_FORNECEDOR_DUPLICADO','CADASTRO_ITEM_INATIVO_COM_DEMANDA_ABERTA',
'COMPRAS_REQUISICAO_AGUARDANDO_APROVACAO','COMPRAS_COTACAO_SEM_RESPOSTA','COMPRAS_PEDIDO_ATRASADO','COMPRAS_ITEM_SEM_FORNECEDOR_VALIDO','COMPRAS_PRECO_VENCIDO_VARIACAO','COMPRAS_LAUDO_AUSENTE_RECEBIMENTO','COMPRAS_DOCUMENTO_FORNECEDOR_VENCIMENTO','COMPRAS_RECEBIMENTO_DIVERGENCIA','COMPRAS_LEAD_TIME_ACIMA_CONTRATADO',
'PRODUCAO_OF_ATRASADA_PARADA','PRODUCAO_OPERACAO_SEM_APONTAMENTO','PRODUCAO_CONSUMO_REFUGO_ACIMA_TOLERANCIA','PRODUCAO_FALTA_MATERIAL_OF_LIBERADA','APS_CAPACIDADE_SOBRECARREGADA','APS_PROMESSA_ENTREGA_EM_RISCO','MANUTENCAO_PREVENTIVA_VENCIDA_PROXIMA','FERRAMENTA_VIDA_UTIL_PROXIMA_EXCEDIDA','CALENDARIO_INDUSTRIAL_INCOMPLETO',
'QUALIDADE_INSPECAO_RECEBIMENTO_PENDENTE','QUALIDADE_NAO_CONFORMIDADE_CRITICA','QUALIDADE_LAUDO_CERTIFICADO_AUSENTE','QUALIDADE_PLANO_OBRIGATORIO_NAO_EXECUTADO','QUALIDADE_LOTE_BLOQUEADO_REPROVADO','QUALIDADE_REINCIDENCIA_ACIMA_LIMITE',
'FINANCEIRO_TITULO_PROXIMO_VENCIMENTO','FINANCEIRO_TITULO_VENCIDO','FINANCEIRO_BLOQUEIO_LIMITE_CREDITO','FINANCEIRO_CONCILIACAO_PENDENTE','FINANCEIRO_REMESSA_RETORNO_REJEITADO','FINANCEIRO_FLUXO_CAIXA_ABAIXO_LIMITE',
'OPERACAO_INTEGRACAO_FISCAL_FALHA','OPERACAO_BACKLOG_DEAD_LETTERS_ANORMAL','OPERACAO_JOB_ESSENCIAL_SEM_EXECUCAO','OPERACAO_IMPORTACAO_REJEITADA_REPETIDAMENTE','OPERACAO_ATUALIZACAO_BACKUP_FALHOU');

CREATE OR REPLACE FUNCTION notification_sync_configured_item(p_item_code BIGINT) RETURNS VOID AS $$
DECLARE
    item_row RECORD;
    has_active_question BOOLEAN;
    next_cycle INTEGER;
BEGIN
    SELECT id, enterprise_id, business_code, pdm_description_technique, engineering_item_base_cod, nature, created_by
      INTO item_row FROM items WHERE code=p_item_code;
    IF NOT FOUND OR item_row.nature<>1 THEN RETURN; END IF;
    SELECT EXISTS(SELECT 1 FROM cfg_item_characteristics ic JOIN cfg_characteristics c ON c.id=ic.characteristic_id WHERE ic.item_code=p_item_code AND c.is_active)
      INTO has_active_question;
    IF has_active_question THEN
        UPDATE notification_alerts SET state='RESOLVIDO',resolved_at=NOW(),resolution_reason='Primeira característica ativa associada'
         WHERE enterprise_id=item_row.enterprise_id AND event_key='CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS' AND aggregate_internal_id=item_row.id::text AND state='ABERTO';
        UPDATE notification_outbox SET state='CANCELADO',processed_at=NOW(),lease_owner=NULL,lease_until=NULL
         WHERE enterprise_id=item_row.enterprise_id AND event_key='CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS' AND aggregate_internal_id=item_row.id::text AND state IN ('PENDENTE','FALHOU');
        RETURN;
    END IF;
    IF EXISTS(SELECT 1 FROM notification_alerts WHERE enterprise_id=item_row.enterprise_id AND event_key='CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS' AND aggregate_internal_id=item_row.id::text AND state='ABERTO') THEN RETURN; END IF;
    SELECT COALESCE(MAX(cycle),0)+1 INTO next_cycle FROM notification_alerts WHERE enterprise_id=item_row.enterprise_id AND event_key='CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS' AND aggregate_internal_id=item_row.id::text;
    INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key,originator_user_id)
    VALUES(item_row.enterprise_id,'CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS',1,'ITEM',item_row.id::text,item_row.business_code,
      jsonb_build_object('codigo',item_row.business_code,'descricao',item_row.pdm_description_technique,'item_base',item_row.engineering_item_base_cod,'criador_usuario_id',item_row.created_by,'data',NOW(),'link','/items/'||item_row.business_code),
      'item:'||item_row.id::text||':ciclo:'||next_cycle::text,item_row.created_by)
    ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION notification_item_configured_trigger() RETURNS TRIGGER AS $$
BEGIN PERFORM notification_sync_configured_item(NEW.code); RETURN NEW; END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_notification_item_configured AFTER INSERT OR UPDATE OF nature,enterprise_id ON items FOR EACH ROW EXECUTE FUNCTION notification_item_configured_trigger();

CREATE OR REPLACE FUNCTION notification_item_characteristic_trigger() RETURNS TRIGGER AS $$
BEGIN PERFORM notification_sync_configured_item(COALESCE(NEW.item_code,OLD.item_code)); RETURN COALESCE(NEW,OLD); END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_notification_item_characteristic AFTER INSERT OR UPDATE OR DELETE ON cfg_item_characteristics FOR EACH ROW EXECUTE FUNCTION notification_item_characteristic_trigger();

CREATE OR REPLACE FUNCTION notification_characteristic_active_trigger() RETURNS TRIGGER AS $$
DECLARE item_code_row BIGINT; BEGIN
  IF OLD.is_active IS DISTINCT FROM NEW.is_active THEN FOR item_code_row IN SELECT item_code FROM cfg_item_characteristics WHERE characteristic_id=NEW.id LOOP PERFORM notification_sync_configured_item(item_code_row); END LOOP; END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_notification_characteristic_active AFTER UPDATE OF is_active ON cfg_characteristics FOR EACH ROW EXECUTE FUNCTION notification_characteristic_active_trigger();
