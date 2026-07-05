CREATE TABLE IF NOT EXISTS technical_assistance_defect_groups (
    code BIGSERIAL PRIMARY KEY,
    description VARCHAR(120) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL
);

CREATE TABLE IF NOT EXISTS technical_assistance_defect_reasons (
    code BIGSERIAL PRIMARY KEY,
    group_code BIGINT NOT NULL REFERENCES technical_assistance_defect_groups(code),
    description VARCHAR(160) NOT NULL,
    allows_complement BOOLEAN NOT NULL DEFAULT FALSE,
    generates_revenue BOOLEAN NOT NULL DEFAULT FALSE,
    requires_return_note BOOLEAN NOT NULL DEFAULT TRUE,
    generates_sales_order BOOLEAN NOT NULL DEFAULT FALSE,
    generates_production_order BOOLEAN NOT NULL DEFAULT TRUE,
    is_replacement BOOLEAN NOT NULL DEFAULT FALSE,
    is_service BOOLEAN NOT NULL DEFAULT FALSE,
    available_web BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL
);

CREATE TABLE IF NOT EXISTS technical_assistance_warranty_responsibles (
    code BIGSERIAL PRIMARY KEY,
    name VARCHAR(160) NOT NULL,
    employee_code BIGINT,
    customer_code BIGINT,
    email VARCHAR(160),
    phone VARCHAR(40),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    CONSTRAINT ta_warranty_responsible_subject_chk
        CHECK (employee_code IS NOT NULL OR customer_code IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS technical_assistance_calls (
    code BIGSERIAL PRIMARY KEY,
    call_number BIGINT NOT NULL,
    enterprise_code BIGINT NOT NULL,
    customer_code BIGINT NOT NULL,
    consumer_name VARCHAR(160),
    consumer_document VARCHAR(32),
    technical_assistant_code BIGINT,
    warranty_responsible_code BIGINT REFERENCES technical_assistance_warranty_responsibles(code),
    status VARCHAR(24) NOT NULL DEFAULT 'PENDING',
    priority VARCHAR(16) NOT NULL DEFAULT 'NORMAL',
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    promised_date DATE,
    attended_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    subject VARCHAR(180) NOT NULL,
    description TEXT,
    diagnosis TEXT,
    solution TEXT,
    return_note_required BOOLEAN NOT NULL DEFAULT TRUE,
    sales_order_code BIGINT,
    production_order_id BIGINT,
    service_invoice_number VARCHAR(80),
    close_reason TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    UNIQUE (enterprise_code, call_number),
    CONSTRAINT ta_call_status_chk CHECK (status IN ('PENDING','IN_ANALYSIS','WAITING_RETURN','WAITING_ORDER','ATTENDED','CLOSED','CANCELLED')),
    CONSTRAINT ta_call_priority_chk CHECK (priority IN ('LOW','NORMAL','HIGH','CRITICAL'))
);

CREATE INDEX IF NOT EXISTS idx_ta_calls_customer ON technical_assistance_calls(customer_code);
CREATE INDEX IF NOT EXISTS idx_ta_calls_status ON technical_assistance_calls(status);
CREATE INDEX IF NOT EXISTS idx_ta_calls_opened_at ON technical_assistance_calls(opened_at);

CREATE TABLE IF NOT EXISTS technical_assistance_call_items (
    code BIGSERIAL PRIMARY KEY,
    call_code BIGINT NOT NULL REFERENCES technical_assistance_calls(code) ON DELETE CASCADE,
    sequence INTEGER NOT NULL,
    item_code BIGINT NOT NULL,
    mask VARCHAR(120) NOT NULL DEFAULT '',
    serial_number VARCHAR(120),
    quantity NUMERIC(15,4) NOT NULL DEFAULT 1,
    defect_reason_code BIGINT REFERENCES technical_assistance_defect_reasons(code),
    defect_complement TEXT,
    purchase_invoice_number VARCHAR(80),
    purchase_invoice_date DATE,
    warranty_days INTEGER NOT NULL DEFAULT 0,
    warranty_until DATE,
    in_warranty BOOLEAN NOT NULL DEFAULT FALSE,
    generates_revenue BOOLEAN NOT NULL DEFAULT FALSE,
    requested_action VARCHAR(24) NOT NULL DEFAULT 'REPAIR',
    status VARCHAR(24) NOT NULL DEFAULT 'OPEN',
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (call_code, sequence),
    CONSTRAINT ta_call_item_qty_chk CHECK (quantity > 0),
    CONSTRAINT ta_call_item_action_chk CHECK (requested_action IN ('REPAIR','REPLACE','RETURN','SERVICE','INSPECT')),
    CONSTRAINT ta_call_item_status_chk CHECK (status IN ('OPEN','IN_PROGRESS','ATTENDED','CANCELLED'))
);

CREATE TABLE IF NOT EXISTS technical_assistance_return_notes (
    code BIGSERIAL PRIMARY KEY,
    call_code BIGINT NOT NULL REFERENCES technical_assistance_calls(code) ON DELETE CASCADE,
    note_number VARCHAR(80) NOT NULL,
    note_series VARCHAR(20),
    emission_date DATE NOT NULL,
    customer_code BIGINT,
    operation_type VARCHAR(24) NOT NULL DEFAULT 'RETURN',
    access_key VARCHAR(80),
    total_value NUMERIC(15,2) NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    CONSTRAINT ta_return_note_operation_chk CHECK (operation_type IN ('RETURN','SHIPMENT','SERVICE_CHARGE'))
);

CREATE TABLE IF NOT EXISTS technical_assistance_order_links (
    code BIGSERIAL PRIMARY KEY,
    call_code BIGINT NOT NULL REFERENCES technical_assistance_calls(code) ON DELETE CASCADE,
    call_item_code BIGINT REFERENCES technical_assistance_call_items(code) ON DELETE SET NULL,
    generated_type VARCHAR(24) NOT NULL,
    sales_order_code BIGINT,
    production_order_id BIGINT,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    notes TEXT,
    CONSTRAINT ta_order_link_type_chk CHECK (generated_type IN ('SALES_ORDER','PRODUCTION_ORDER'))
);
