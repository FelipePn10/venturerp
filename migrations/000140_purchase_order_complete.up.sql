BEGIN;

-- ─── Pedido de Compra completo — campos de capa ───────────────────────────────
ALTER TABLE public.purchase_orders
    ADD COLUMN IF NOT EXISTS price_table_code         BIGINT,
    ADD COLUMN IF NOT EXISTS invoice_type_code        BIGINT,
    ADD COLUMN IF NOT EXISTS financial_account        VARCHAR(30),
    ADD COLUMN IF NOT EXISTS request_type_code        BIGINT,
    ADD COLUMN IF NOT EXISTS currency_date            DATE,
    ADD COLUMN IF NOT EXISTS freight_type             VARCHAR(15) NOT NULL DEFAULT 'SEM_FRETE',
    ADD COLUMN IF NOT EXISTS freight_value_type       VARCHAR(12),   -- VALOR | PERCENTUAL
    ADD COLUMN IF NOT EXISTS freight_value_mode       VARCHAR(12),   -- UNITARIO | TOTAL
    ADD COLUMN IF NOT EXISTS freight_value            NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS carrier_code             BIGINT,
    ADD COLUMN IF NOT EXISTS redispatch_carrier_code  BIGINT,
    ADD COLUMN IF NOT EXISTS redispatch_freight_type  VARCHAR(15),
    ADD COLUMN IF NOT EXISTS redispatch_freight_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS advance_date             DATE,
    ADD COLUMN IF NOT EXISTS advance_value            NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS incoterm_code            VARCHAR(10),
    ADD COLUMN IF NOT EXISTS shipment_date            DATE,
    ADD COLUMN IF NOT EXISTS talao_number             VARCHAR(30),
    ADD COLUMN IF NOT EXISTS alcada_status            VARCHAR(2) NOT NULL DEFAULT 'N'; -- A/B/R/I/N

-- ─── Pedido de Compra completo — campos de item ───────────────────────────────
ALTER TABLE public.purchase_order_items
    ADD COLUMN IF NOT EXISTS purchase_uom              VARCHAR(10),
    ADD COLUMN IF NOT EXISTS internal_uom              VARCHAR(10),
    ADD COLUMN IF NOT EXISTS internal_qty              NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS internal_price            NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS promised_date             DATE,
    ADD COLUMN IF NOT EXISTS tolerance_pct             NUMERIC(7,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cancelled_tolerance_qty   NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS icms_st_pct               NUMERIC(7,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS operation_type_code       BIGINT,
    ADD COLUMN IF NOT EXISTS invoice_type_code         BIGINT,
    ADD COLUMN IF NOT EXISTS accounting_account        VARCHAR(30),
    ADD COLUMN IF NOT EXISTS cost_center_code          BIGINT,
    ADD COLUMN IF NOT EXISTS requester_employee_code   BIGINT,
    ADD COLUMN IF NOT EXISTS contract_code             BIGINT,
    ADD COLUMN IF NOT EXISTS quotation_code            BIGINT,
    ADD COLUMN IF NOT EXISTS utilization_type          VARCHAR(20), -- INDUSTRIALIZACAO | CONSUMO | IMOBILIZADO
    ADD COLUMN IF NOT EXISTS fiscal_classification_code BIGINT;

COMMIT;
