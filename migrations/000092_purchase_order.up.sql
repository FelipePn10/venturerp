BEGIN;
CREATE TABLE IF NOT EXISTS public.purchase_orders (
    code                    BIGSERIAL PRIMARY KEY,
    order_number            BIGINT NOT NULL,
    enterprise_code         BIGINT NOT NULL,
    status                  VARCHAR(10) NOT NULL DEFAULT 'DRAFT',
    origin                  VARCHAR(20) NOT NULL DEFAULT 'NORMAL',
    emission_date           DATE NOT NULL DEFAULT CURRENT_DATE,
    delivery_date           DATE,
    supplier_code           BIGINT,
    payment_term_code       BIGINT,
    currency_code           VARCHAR(5) NOT NULL DEFAULT 'BRL',
    shipping_address_code   BIGINT,
    notes                   TEXT,
    
    total_gross             NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_net               NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_discount          NUMERIC(15,4) NOT NULL DEFAULT 0,
    
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    is_firm                 BOOLEAN NOT NULL DEFAULT FALSE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL
);

CREATE TABLE IF NOT EXISTS public.purchase_order_items (
    code                    BIGSERIAL PRIMARY KEY,
    purchase_order_code     BIGINT NOT NULL REFERENCES public.purchase_orders(code),
    sequence                INT NOT NULL DEFAULT 1,
    item_code               BIGINT NOT NULL,
    mask                    VARCHAR(200) NOT NULL DEFAULT '',
    requested_qty           NUMERIC(15,4) NOT NULL DEFAULT 0,
    received_qty            NUMERIC(15,4) NOT NULL DEFAULT 0,
    cancelled_qty           NUMERIC(15,4) NOT NULL DEFAULT 0,
    unit_price              NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_price             NUMERIC(15,4) NOT NULL DEFAULT 0,
    discount_pct            NUMERIC(7,4) NOT NULL DEFAULT 0,
    ipi_pct                 NUMERIC(7,4) NOT NULL DEFAULT 0,
    icms_pct                NUMERIC(7,4) NOT NULL DEFAULT 0,
    status                  VARCHAR(20) NOT NULL DEFAULT 'OPEN',
    delivery_date           DATE,
    notes                   TEXT,
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.purchase_order_sequences (
    enterprise_code BIGINT PRIMARY KEY,
    last_number     BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_po_supplier ON public.purchase_orders(supplier_code);
CREATE INDEX idx_po_status ON public.purchase_orders(status);
CREATE INDEX idx_po_items_order ON public.purchase_order_items(purchase_order_code);
CREATE INDEX idx_po_items_item ON public.purchase_order_items(item_code);
COMMIT;
