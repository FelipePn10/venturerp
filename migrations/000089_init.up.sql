BEGIN;

-- =========================================================
-- SALES ORDERS (PEDIDOS DE VENDA)
-- =========================================================

CREATE TABLE IF NOT EXISTS public.sales_orders (
    code                    BIGSERIAL PRIMARY KEY,
    order_number            BIGINT        NOT NULL,
    enterprise_code         BIGINT        NOT NULL,
    status                  VARCHAR(5)    NOT NULL DEFAULT 'R',   -- R, P, A, OA, OF
    origin                  VARCHAR(20)   NOT NULL DEFAULT 'NORMAL', -- NORMAL, DEPENDENT, ASSISTANCE, RESERVE, COPY
    emission_date           DATE          NOT NULL DEFAULT CURRENT_DATE,
    delivery_date           DATE,
    delivery_date_firm      BOOLEAN       NOT NULL DEFAULT FALSE,
    digit_date              DATE          NOT NULL DEFAULT CURRENT_DATE,
    customer_code           BIGINT,
    billing_address_code    BIGINT,
    shipping_address_code   BIGINT,
    representative_code     BIGINT,
    plan_code               BIGINT        REFERENCES production_plans(code),
    sales_division_code     BIGINT        REFERENCES sales_divisions(id),
    commission_pct          NUMERIC(7,4)  NOT NULL DEFAULT 0,
    tax_type_code           BIGINT,
    presence_indicator      VARCHAR(5),
    sales_channel           VARCHAR(50),
    default_nf_type         VARCHAR(5),
    price_table_code        BIGINT,
    currency_code           VARCHAR(5)    NOT NULL DEFAULT 'BRL',
    payment_term_code       BIGINT,
    additional_days         INT           NOT NULL DEFAULT 0,
    bearer_code             BIGINT,
    sale_date               DATE,

    -- Weight totals
    total_weight_net        NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_weight_gross      NUMERIC(15,4) NOT NULL DEFAULT 0,

    -- Monetary totals
    total_gross             NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_net               NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_net_no_st         NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_with_ipi_with_st  NUMERIC(15,4) NOT NULL DEFAULT 0,

    -- Control
    notes                   TEXT,
    obs_customer            TEXT,
    is_blocked              BOOLEAN       NOT NULL DEFAULT FALSE,
    block_reason            TEXT,
    is_firm                 BOOLEAN       NOT NULL DEFAULT FALSE,
    is_active               BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by              UUID          NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sales_orders_number_enterprise
    ON public.sales_orders (order_number, enterprise_code);

CREATE INDEX IF NOT EXISTS idx_sales_orders_customer
    ON public.sales_orders (customer_code);

CREATE INDEX IF NOT EXISTS idx_sales_orders_status
    ON public.sales_orders (status);

CREATE INDEX IF NOT EXISTS idx_sales_orders_delivery_date
    ON public.sales_orders (delivery_date);

CREATE INDEX IF NOT EXISTS idx_sales_orders_emission_date
    ON public.sales_orders (emission_date);

-- =========================================================
-- SALES ORDER ITEMS (ITENS DO PEDIDO DE VENDA)
-- =========================================================

CREATE TABLE IF NOT EXISTS public.sales_order_items (
    code                    BIGSERIAL PRIMARY KEY,
    sales_order_code        BIGINT        NOT NULL REFERENCES public.sales_orders(code),
    sequence                INT           NOT NULL DEFAULT 1,
    item_code               BIGINT        NOT NULL,
    mask                    VARCHAR(200)  NOT NULL DEFAULT '',
    digit_date              DATE          NOT NULL DEFAULT CURRENT_DATE,
    nf_type                 VARCHAR(5),
    sales_uom               VARCHAR(10),
    warehouse_code          BIGINT,
    price_table_code        BIGINT,

    -- Quantities
    requested_qty           NUMERIC(15,4) NOT NULL DEFAULT 0,
    unit_price              NUMERIC(15,4) NOT NULL DEFAULT 0,
    attended_qty            NUMERIC(15,4) NOT NULL DEFAULT 0,
    cancelled_qty           NUMERIC(15,4) NOT NULL DEFAULT 0,

    -- Delivery
    delivery_date           DATE,
    delivery_date_firm      BOOLEAN       NOT NULL DEFAULT FALSE,
    customer_delivery       VARCHAR(100),
    lot                     VARCHAR(50),
    coupon_delivery         VARCHAR(50),
    paid_at_cashier         BOOLEAN       NOT NULL DEFAULT FALSE,

    -- Financial values
    ipi_pct                 NUMERIC(7,4)  NOT NULL DEFAULT 0,
    icms_pct                NUMERIC(7,4)  NOT NULL DEFAULT 0,
    pis_pct                 NUMERIC(7,4)  NOT NULL DEFAULT 0,
    cofins_pct              NUMERIC(7,4)  NOT NULL DEFAULT 0,
    st_pct                  NUMERIC(7,4)  NOT NULL DEFAULT 0,
    discount_pct            NUMERIC(7,4)  NOT NULL DEFAULT 0,

    total_gross             NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_net               NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_net_with_ipi      NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_ipi               NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_st                NUMERIC(15,4) NOT NULL DEFAULT 0,

    -- Weight
    unit_weight_net         NUMERIC(15,4) NOT NULL DEFAULT 0,
    unit_weight_gross       NUMERIC(15,4) NOT NULL DEFAULT 0,

    -- Control
    status                  VARCHAR(20)   NOT NULL DEFAULT 'OPEN', -- OPEN, PARTIAL, DELIVERED, CANCELLED
    notes                   TEXT,
    is_active               BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sales_order_items_order
    ON public.sales_order_items (sales_order_code);

CREATE INDEX IF NOT EXISTS idx_sales_order_items_item
    ON public.sales_order_items (item_code);

CREATE INDEX IF NOT EXISTS idx_sales_order_items_delivery
    ON public.sales_order_items (delivery_date);

-- =========================================================
-- SEQUENCE FOR ORDER NUMBER PER ENTERPRISE
-- =========================================================

CREATE TABLE IF NOT EXISTS public.sales_order_sequences (
    enterprise_code BIGINT PRIMARY KEY,
    last_number     BIGINT NOT NULL DEFAULT 0
);

COMMIT;
