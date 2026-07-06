CREATE TABLE IF NOT EXISTS consumer_service_call_types (
    code BIGSERIAL PRIMARY KEY,
    description VARCHAR(120) NOT NULL,
    is_complaint BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL
);

CREATE TABLE IF NOT EXISTS consumer_service_knowledge_sources (
    code BIGSERIAL PRIMARY KEY,
    description VARCHAR(120) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL
);

CREATE TABLE IF NOT EXISTS consumer_service_consumers (
    code BIGINT PRIMARY KEY,
    name VARCHAR(180) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    person_type CHAR(1) NOT NULL DEFAULT 'F',
    cpf VARCHAR(20),
    rg VARCHAR(40),
    cnpj VARCHAR(24),
    state_registration VARCHAR(40),
    zip_code VARCHAR(16),
    city VARCHAR(120),
    state CHAR(2),
    address VARCHAR(180),
    address_number VARCHAR(30),
    complement VARCHAR(120),
    district VARCHAR(120),
    market_segment_code BIGINT,
    knowledge_code BIGINT REFERENCES consumer_service_knowledge_sources(code),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    CONSTRAINT consumer_service_person_type_chk CHECK (person_type IN ('F','J')),
    CONSTRAINT consumer_service_document_chk CHECK (
        (person_type = 'F' AND cnpj IS NULL) OR
        (person_type = 'J' AND cpf IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_consumer_service_consumers_name ON consumer_service_consumers(name);
CREATE INDEX IF NOT EXISTS idx_consumer_service_consumers_state_city ON consumer_service_consumers(state, city);

CREATE TABLE IF NOT EXISTS consumer_service_consumer_contacts (
    code BIGSERIAL PRIMARY KEY,
    consumer_code BIGINT NOT NULL REFERENCES consumer_service_consumers(code) ON DELETE CASCADE,
    name VARCHAR(160) NOT NULL,
    role VARCHAR(80),
    contact_type VARCHAR(40),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS consumer_service_consumer_phones (
    code BIGSERIAL PRIMARY KEY,
    consumer_code BIGINT NOT NULL REFERENCES consumer_service_consumers(code) ON DELETE CASCADE,
    contact_code BIGINT REFERENCES consumer_service_consumer_contacts(code) ON DELETE CASCADE,
    phone_type VARCHAR(24) NOT NULL DEFAULT 'PHONE',
    number VARCHAR(40) NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS consumer_service_consumer_emails (
    code BIGSERIAL PRIMARY KEY,
    consumer_code BIGINT NOT NULL REFERENCES consumer_service_consumers(code) ON DELETE CASCADE,
    contact_code BIGINT REFERENCES consumer_service_consumer_contacts(code) ON DELETE CASCADE,
    email VARCHAR(180) NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS consumer_service_customer_contacts (
    code BIGSERIAL PRIMARY KEY,
    customer_code BIGINT NOT NULL,
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_code BIGINT,
    contact_type VARCHAR(40) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_consumer_service_customer_contacts_customer ON consumer_service_customer_contacts(customer_code);
CREATE INDEX IF NOT EXISTS idx_consumer_service_customer_contacts_opened ON consumer_service_customer_contacts(opened_at);

CREATE TABLE IF NOT EXISTS consumer_service_calls (
    code BIGSERIAL PRIMARY KEY,
    call_number BIGINT NOT NULL,
    enterprise_code BIGINT NOT NULL,
    consumer_code BIGINT NOT NULL REFERENCES consumer_service_consumers(code),
    customer_code BIGINT,
    call_type_code BIGINT NOT NULL REFERENCES consumer_service_call_types(code),
    direction VARCHAR(16) NOT NULL DEFAULT 'RECEIVED',
    in_warranty BOOLEAN NOT NULL DEFAULT FALSE,
    defect_group_code BIGINT,
    defect_reason_code BIGINT,
    responsible_user_code BIGINT,
    position VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    situation VARCHAR(24) NOT NULL DEFAULT 'OTHER',
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    return_date DATE,
    visit_requested_date DATE,
    visit_returned_date DATE,
    sale_store_code BIGINT,
    establishment_code BIGINT,
    technician_description VARCHAR(180),
    symptoms TEXT,
    forwarded_store_code BIGINT,
    subject VARCHAR(180) NOT NULL,
    description TEXT,
    solution TEXT,
    checklist_code BIGINT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    UNIQUE (enterprise_code, call_number),
    CONSTRAINT consumer_service_direction_chk CHECK (direction IN ('RECEIVED','MADE','WARRANTY')),
    CONSTRAINT consumer_service_position_chk CHECK (position IN ('PENDING','SCHEDULED','RESOLVED')),
    CONSTRAINT consumer_service_situation_chk CHECK (situation IN ('OTHER','ORDER','DISCONTINUED_ORDER','TECHNICAL_VISIT')),
    CONSTRAINT consumer_service_visit_date_chk CHECK (situation <> 'TECHNICAL_VISIT' OR visit_requested_date IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_consumer_service_calls_consumer ON consumer_service_calls(consumer_code);
CREATE INDEX IF NOT EXISTS idx_consumer_service_calls_position ON consumer_service_calls(position);
CREATE INDEX IF NOT EXISTS idx_consumer_service_calls_situation ON consumer_service_calls(situation);
CREATE INDEX IF NOT EXISTS idx_consumer_service_calls_opened ON consumer_service_calls(opened_at);
CREATE INDEX IF NOT EXISTS idx_consumer_service_calls_return ON consumer_service_calls(return_date);

CREATE TABLE IF NOT EXISTS consumer_service_call_returns (
    code BIGSERIAL PRIMARY KEY,
    call_code BIGINT NOT NULL REFERENCES consumer_service_calls(code) ON DELETE CASCADE,
    contacted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    contact_type VARCHAR(40) NOT NULL,
    description TEXT NOT NULL,
    next_return_at DATE,
    user_code BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL
);

CREATE TABLE IF NOT EXISTS consumer_service_call_attachments (
    code BIGSERIAL PRIMARY KEY,
    call_code BIGINT NOT NULL REFERENCES consumer_service_calls(code) ON DELETE CASCADE,
    file_name VARCHAR(220) NOT NULL,
    file_path TEXT NOT NULL,
    content_type VARCHAR(120),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL
);

CREATE TABLE IF NOT EXISTS consumer_service_call_checklist_items (
    code BIGSERIAL PRIMARY KEY,
    call_code BIGINT NOT NULL REFERENCES consumer_service_calls(code) ON DELETE CASCADE,
    sequence INTEGER NOT NULL,
    description VARCHAR(180) NOT NULL,
    is_done BOOLEAN NOT NULL DEFAULT FALSE,
    done_at TIMESTAMPTZ,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (call_code, sequence)
);
