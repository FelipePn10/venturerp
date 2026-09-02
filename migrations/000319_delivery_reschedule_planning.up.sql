BEGIN;

ALTER TABLE public.delivery_reschedules
    ADD COLUMN IF NOT EXISTS enterprise_code BIGINT,
    ADD COLUMN IF NOT EXISTS batch_id UUID;

UPDATE public.delivery_reschedules dr
SET enterprise_code = so.enterprise_code
FROM public.sales_orders so
WHERE so.code = dr.sales_order_code AND dr.enterprise_code IS NULL;

-- Bancos demo antigos podem conter histórico anterior ao cadastro de pedidos.
-- O fallback só é seguro quando o banco possui exatamente uma empresa; em um
-- banco multiempresa os NULL restantes fazem o NOT NULL abortar a migration.
UPDATE public.delivery_reschedules
SET enterprise_code = (SELECT MIN(id) FROM public.enterprise)
WHERE enterprise_code IS NULL
  AND (SELECT COUNT(*) FROM public.enterprise) = 1;

ALTER TABLE public.delivery_reschedules ALTER COLUMN enterprise_code SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_sales_orders_tenant_code
    ON public.sales_orders (enterprise_code, code);
ALTER TABLE public.delivery_reschedules
    ADD CONSTRAINT fk_delivery_reschedules_tenant_order
    FOREIGN KEY (enterprise_code, sales_order_code)
    REFERENCES public.sales_orders (enterprise_code, code);

CREATE TABLE public.delivery_reschedule_batches (
    id UUID PRIMARY KEY,
    enterprise_code BIGINT NOT NULL,
    sales_order_code BIGINT NOT NULL,
    idempotency_key VARCHAR(100) NOT NULL,
    payload_hash VARCHAR(64) NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE (enterprise_code, idempotency_key)
);

ALTER TABLE public.delivery_reschedule_batches
    ADD CONSTRAINT fk_delivery_reschedule_batches_tenant_order
    FOREIGN KEY (enterprise_code, sales_order_code)
    REFERENCES public.sales_orders (enterprise_code, code);
CREATE UNIQUE INDEX ux_delivery_reschedule_batches_tenant_id
    ON public.delivery_reschedule_batches (enterprise_code, id);
ALTER TABLE public.delivery_reschedules
    ADD CONSTRAINT fk_delivery_reschedules_tenant_batch
    FOREIGN KEY (enterprise_code, batch_id)
    REFERENCES public.delivery_reschedule_batches (enterprise_code, id);

CREATE INDEX idx_delivery_reschedules_tenant_order
    ON public.delivery_reschedules (enterprise_code, sales_order_code, created_at DESC);
CREATE INDEX idx_delivery_reschedules_batch ON public.delivery_reschedules (batch_id);

COMMIT;
