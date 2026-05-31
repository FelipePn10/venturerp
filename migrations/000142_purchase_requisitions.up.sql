BEGIN;

-- ─── Solicitação de Compra ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS purchase_requisitions (
    id                      BIGSERIAL PRIMARY KEY,
    code                    BIGINT      NOT NULL UNIQUE,
    enterprise_code         BIGINT      NOT NULL,
    request_type_code       BIGINT,
    requester_employee_code BIGINT,
    emission_date           DATE        NOT NULL DEFAULT CURRENT_DATE,
    status                  VARCHAR(12) NOT NULL DEFAULT 'OPEN'
                            CHECK (status IN ('OPEN','PARTIAL','ATTENDED','CANCELLED')),
    notes                   TEXT,
    is_active               BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID        NOT NULL,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS purchase_requisition_items (
    id                 BIGSERIAL PRIMARY KEY,
    requisition_code   BIGINT        NOT NULL REFERENCES purchase_requisitions(code) ON DELETE CASCADE,
    sequence           INT           NOT NULL DEFAULT 1,
    item_code          BIGINT        NOT NULL,
    quantity           NUMERIC(15,4) NOT NULL DEFAULT 0,
    attended_qty       NUMERIC(15,4) NOT NULL DEFAULT 0,
    cancelled_qty      NUMERIC(15,4) NOT NULL DEFAULT 0,
    uom                VARCHAR(10),
    cost_center_code   BIGINT,
    accounting_account VARCHAR(30),
    suggested_price    NUMERIC(15,4) NOT NULL DEFAULT 0,
    delivery_date      DATE,
    application        VARCHAR(200),
    utilization_type   VARCHAR(20),
    status             VARCHAR(12)   NOT NULL DEFAULT 'OPEN'
                       CHECK (status IN ('OPEN','PARTIAL','ATTENDED','CANCELLED')),
    is_active          BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_purchase_requisition_items_req ON purchase_requisition_items(requisition_code);
CREATE INDEX IF NOT EXISTS ix_purchase_requisition_items_open
    ON purchase_requisition_items(item_code) WHERE status IN ('OPEN','PARTIAL') AND is_active = TRUE;

COMMIT;
