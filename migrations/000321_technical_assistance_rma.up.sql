CREATE TABLE IF NOT EXISTS public.technical_assistance_rmas (
    code BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES public.enterprise(id),
    call_code BIGINT NOT NULL REFERENCES public.technical_assistance_calls(code),
    status VARCHAR(32) NOT NULL DEFAULT 'SOLICITADO',
    reason_code VARCHAR(60) NOT NULL,
    reason_description TEXT,
    eligibility_status VARCHAR(20) NOT NULL,
    eligibility_reason TEXT NOT NULL,
    authorization_number VARCHAR(80),
    authorized_at TIMESTAMPTZ,
    reverse_carrier_code BIGINT,
    reverse_tracking_code VARCHAR(120),
    received_at TIMESTAMPTZ,
    inspection_notes TEXT,
    inspected_at TIMESTAMPTZ,
    destination VARCHAR(20),
    sla_due_at TIMESTAMPTZ NOT NULL,
    product_cost NUMERIC(15,4) NOT NULL DEFAULT 0,
    freight_cost NUMERIC(15,4) NOT NULL DEFAULT 0,
    service_cost NUMERIC(15,4) NOT NULL DEFAULT 0,
    fiscal_document_key VARCHAR(60),
    stock_movement_code BIGINT,
    idempotency_key VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    CONSTRAINT technical_assistance_rma_status_chk CHECK (status IN ('SOLICITADO','AUTORIZADO','LOGISTICA_REVERSA','RECEBIDO','INSPECIONADO','RESOLVIDO','CANCELADO')),
    CONSTRAINT technical_assistance_rma_eligibility_chk CHECK (eligibility_status IN ('ELEGIVEL','NAO_ELEGIVEL','REVISAO')),
    CONSTRAINT technical_assistance_rma_destination_chk CHECK (destination IS NULL OR destination IN ('REPARO','TROCA','CREDITO','SUCATA')),
    CONSTRAINT technical_assistance_rma_cost_chk CHECK (product_cost >= 0 AND freight_cost >= 0 AND service_cost >= 0),
    UNIQUE (enterprise_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_technical_assistance_rma_call
    ON public.technical_assistance_rmas (enterprise_id, call_code, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_technical_assistance_rma_sla
    ON public.technical_assistance_rmas (enterprise_id, status, sla_due_at);

CREATE OR REPLACE FUNCTION public.validate_technical_assistance_rma_tenant()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM public.technical_assistance_calls c
          JOIN public.enterprise e ON e.code = c.enterprise_code
         WHERE c.code = NEW.call_code AND e.id = NEW.enterprise_id
    ) THEN
        RAISE EXCEPTION 'RMA e chamado devem pertencer à mesma empresa';
    END IF;
    RETURN NEW;
END
$$;
CREATE TRIGGER trg_technical_assistance_rma_tenant
BEFORE INSERT OR UPDATE OF enterprise_id, call_code ON public.technical_assistance_rmas
FOR EACH ROW EXECUTE FUNCTION public.validate_technical_assistance_rma_tenant();

CREATE TABLE IF NOT EXISTS public.technical_assistance_rma_items (
    code BIGSERIAL PRIMARY KEY,
    rma_code BIGINT NOT NULL REFERENCES public.technical_assistance_rmas(code) ON DELETE CASCADE,
    call_item_code BIGINT NOT NULL REFERENCES public.technical_assistance_call_items(code),
    item_code BIGINT NOT NULL,
    quantity NUMERIC(15,4) NOT NULL,
    serial_number VARCHAR(120),
    lot_number VARCHAR(120),
    requested_destination VARCHAR(20),
    inspection_result VARCHAR(30),
    CONSTRAINT technical_assistance_rma_item_qty_chk CHECK (quantity > 0),
    CONSTRAINT technical_assistance_rma_item_destination_chk CHECK (requested_destination IS NULL OR requested_destination IN ('REPARO','TROCA','CREDITO','SUCATA')),
    UNIQUE (rma_code, call_item_code, serial_number, lot_number)
);

CREATE TABLE IF NOT EXISTS public.technical_assistance_rma_events (
    code BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES public.enterprise(id),
    rma_code BIGINT NOT NULL REFERENCES public.technical_assistance_rmas(code) ON DELETE CASCADE,
    event_type VARCHAR(40) NOT NULL,
    before_state JSONB,
    after_state JSONB NOT NULL,
    reason TEXT,
    correlation_id VARCHAR(100),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_id UUID NOT NULL
);

CREATE UNIQUE INDEX ux_technical_assistance_rmas_tenant_code
    ON public.technical_assistance_rmas (enterprise_id, code);
ALTER TABLE public.technical_assistance_rma_events
    DROP CONSTRAINT technical_assistance_rma_events_rma_code_fkey,
    ADD CONSTRAINT fk_technical_assistance_rma_events_tenant_rma
    FOREIGN KEY (enterprise_id, rma_code)
    REFERENCES public.technical_assistance_rmas (enterprise_id, code) ON DELETE CASCADE;

CREATE OR REPLACE FUNCTION public.validate_technical_assistance_rma_item_scope()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM public.technical_assistance_rmas r
          JOIN public.technical_assistance_call_items ci ON ci.code = NEW.call_item_code
         WHERE r.code = NEW.rma_code AND ci.call_code = r.call_code
    ) THEN
        RAISE EXCEPTION 'item do RMA não pertence ao chamado associado';
    END IF;
    RETURN NEW;
END
$$;
CREATE TRIGGER trg_technical_assistance_rma_item_scope
BEFORE INSERT OR UPDATE OF rma_code, call_item_code ON public.technical_assistance_rma_items
FOR EACH ROW EXECUTE FUNCTION public.validate_technical_assistance_rma_item_scope();

CREATE INDEX IF NOT EXISTS idx_technical_assistance_rma_events
    ON public.technical_assistance_rma_events (enterprise_id, rma_code, occurred_at, code);
