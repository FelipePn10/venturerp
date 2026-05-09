CREATE TABLE IF NOT EXISTS mrp_exception_messages (
    code         BIGSERIAL    PRIMARY KEY,
    plan_code    BIGINT       NOT NULL,
    item_code    BIGINT       NOT NULL,
    message_type VARCHAR(30)  NOT NULL,  -- RESCHEDULE_IN | RESCHEDULE_OUT | CANCEL | EXPEDITE | EXCESS_PROJECTED
    source_code  BIGINT,                 -- PK of the planned_order that triggered this (nullable)
    source_type  VARCHAR(30),            -- e.g. PLANNED_ORDER, PURCHASE_ORDER
    description  TEXT         NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mrp_exceptions_plan  ON mrp_exception_messages (plan_code);
CREATE INDEX IF NOT EXISTS idx_mrp_exceptions_item  ON mrp_exception_messages (item_code);
CREATE INDEX IF NOT EXISTS idx_mrp_exceptions_type  ON mrp_exception_messages (message_type);
