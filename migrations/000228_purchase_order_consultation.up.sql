BEGIN;

ALTER TABLE public.purchase_orders
    ADD COLUMN IF NOT EXISTS buyer_employee_code BIGINT,
    ADD COLUMN IF NOT EXISTS order_type VARCHAR(3) NOT NULL DEFAULT 'OCL',
    ADD COLUMN IF NOT EXISTS kanban_origin BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS customer_code BIGINT;

ALTER TABLE public.purchase_orders
    ADD CONSTRAINT purchase_orders_order_type_chk
    CHECK (order_type IN ('OCL','OSL','ORM','ORD'));

ALTER TABLE public.purchase_order_items
    ADD COLUMN IF NOT EXISTS additions NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS ipi_base NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS ipi_value NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS icms_base NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS icms_value NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS icms_st_base NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS icms_st_value NUMERIC(15,4);

CREATE TABLE public.purchase_order_currency_rates (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES public.enterprise(id) ON DELETE CASCADE,
    currency_code VARCHAR(3) NOT NULL,
    rate_date DATE NOT NULL,
    rate_to_base NUMERIC(20,8) NOT NULL CHECK (rate_to_base > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, currency_code, rate_date)
);

CREATE TABLE public.purchase_order_attachments (
    id BIGSERIAL PRIMARY KEY,
    purchase_order_code BIGINT NOT NULL REFERENCES public.purchase_orders(code) ON DELETE CASCADE,
    file_name TEXT NOT NULL CHECK (btrim(file_name) <> ''),
    content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
    content BYTEA NOT NULL,
    file_size BIGINT GENERATED ALWAYS AS (octet_length(content)) STORED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    CHECK (octet_length(content) <= 10485760)
);

CREATE INDEX idx_purchase_orders_consultation
    ON public.purchase_orders (enterprise_code, order_number, supplier_code, emission_date, delivery_date);
CREATE INDEX idx_purchase_orders_consultation_metadata
    ON public.purchase_orders (enterprise_code, request_type_code, buyer_employee_code, order_type, kanban_origin);
CREATE INDEX idx_purchase_order_items_consultation
    ON public.purchase_order_items (purchase_order_code, item_code, is_active);
CREATE INDEX idx_import_processes_purchase_order
    ON public.import_processes (enterprise_code, purchase_order_code, process_number);
CREATE INDEX idx_purchase_order_attachments_order
    ON public.purchase_order_attachments (purchase_order_code, created_at DESC);

COMMIT;
