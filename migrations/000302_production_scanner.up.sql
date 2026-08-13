BEGIN;
CREATE TABLE production_scan_tokens (
 id BIGSERIAL PRIMARY KEY, enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
 production_order_id BIGINT NOT NULL REFERENCES production_orders(id) ON DELETE CASCADE,
 operation_id BIGINT REFERENCES production_order_operations(id) ON DELETE CASCADE,
 token_hash BYTEA NOT NULL, active BOOLEAN NOT NULL DEFAULT TRUE,
 valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(), valid_until TIMESTAMPTZ,
 created_by UUID NOT NULL REFERENCES users(id), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), revoked_at TIMESTAMPTZ,
 CONSTRAINT uq_production_scan_token_hash UNIQUE(token_hash),
 CONSTRAINT chk_production_scan_token_validity CHECK(valid_until IS NULL OR valid_until>=valid_from)
);
CREATE INDEX idx_production_scan_token_target ON production_scan_tokens(enterprise_id,production_order_id,operation_id) WHERE active;
CREATE TABLE production_scan_events (
 id BIGSERIAL PRIMARY KEY, enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
 token_id BIGINT REFERENCES production_scan_tokens(id), production_order_id BIGINT REFERENCES production_orders(id),
 operation_id BIGINT REFERENCES production_order_operations(id), user_id UUID NOT NULL REFERENCES users(id),
 device_id VARCHAR(120) NOT NULL, action VARCHAR(20) NOT NULL CHECK(action IN ('RESOLVER','INICIAR','APONTAR','CONCLUIR')),
 result VARCHAR(20) NOT NULL CHECK(result IN ('SUCESSO','REJEITADO','ERRO')),
 idempotency_key VARCHAR(120) NOT NULL, request_fingerprint BYTEA NOT NULL,
 response JSONB, message TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 CONSTRAINT uq_production_scan_idempotency UNIQUE(enterprise_id,user_id,action,idempotency_key)
);
CREATE INDEX idx_production_scan_events_audit ON production_scan_events(enterprise_id,created_at DESC);
COMMIT;
