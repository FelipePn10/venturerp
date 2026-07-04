-- Suprimentos close-out: receiving notice + divergences, supplier EDI (structured),
-- import landed cost, procurement parameters and supplier homologation.

-- ---- Receiving notice + dock schedule + divergences (FAVR) ----
CREATE TABLE receiving_notices (
  id BIGSERIAL PRIMARY KEY,
  enterprise_code BIGINT NOT NULL DEFAULT 1,
  notice_number BIGINT GENERATED ALWAYS AS IDENTITY,
  supplier_code BIGINT,
  purchase_order_code BIGINT,
  carrier_code BIGINT,
  status TEXT NOT NULL DEFAULT 'SCHEDULED',   -- SCHEDULED|ARRIVED|IN_CONFERENCE|RELEASED|BLOCKED|CANCELLED
  dock TEXT,
  scheduled_at TIMESTAMPTZ,
  arrived_at TIMESTAMPTZ,
  invoice_number TEXT,
  blocked BOOLEAN NOT NULL DEFAULT FALSE,
  notes TEXT,
  created_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (status IN ('SCHEDULED','ARRIVED','IN_CONFERENCE','RELEASED','BLOCKED','CANCELLED'))
);
CREATE INDEX idx_receiving_notices_status ON receiving_notices(status, scheduled_at);
CREATE INDEX idx_receiving_notices_supplier ON receiving_notices(supplier_code) WHERE supplier_code IS NOT NULL;

CREATE TABLE receiving_notice_items (
  id BIGSERIAL PRIMARY KEY,
  notice_id BIGINT NOT NULL REFERENCES receiving_notices(id) ON DELETE CASCADE,
  purchase_order_item_code BIGINT,
  item_code BIGINT NOT NULL,
  mask TEXT NOT NULL DEFAULT '',
  expected_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  received_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  unit TEXT,
  notes TEXT,
  CHECK (expected_qty >= 0),
  CHECK (received_qty >= 0)
);
CREATE INDEX idx_receiving_notice_items_notice ON receiving_notice_items(notice_id);

CREATE TABLE receiving_divergences (
  id BIGSERIAL PRIMARY KEY,
  notice_id BIGINT REFERENCES receiving_notices(id) ON DELETE CASCADE,
  purchase_order_code BIGINT,
  purchase_order_item_code BIGINT,
  supplier_code BIGINT,
  item_code BIGINT,
  mask TEXT NOT NULL DEFAULT '',
  divergence_type TEXT NOT NULL,             -- SHORTAGE|EXCESS|DAMAGE|WRONG_ITEM|PRICE|DOCUMENT|LATE|OTHER
  expected_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  actual_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  expected_price NUMERIC(15,4),
  actual_price NUMERIC(15,4),
  resolution TEXT NOT NULL DEFAULT 'PENDING', -- PENDING|ACCEPTED|PARTIAL_RETURN|FULL_RETURN|WAIVED|SUPPLIER_DEBIT
  affects_supplier_score BOOLEAN NOT NULL DEFAULT TRUE,
  notes TEXT,
  created_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  resolved_at TIMESTAMPTZ,
  CHECK (divergence_type IN ('SHORTAGE','EXCESS','DAMAGE','WRONG_ITEM','PRICE','DOCUMENT','LATE','OTHER')),
  CHECK (resolution IN ('PENDING','ACCEPTED','PARTIAL_RETURN','FULL_RETURN','WAIVED','SUPPLIER_DEBIT'))
);
CREATE INDEX idx_receiving_divergences_supplier ON receiving_divergences(supplier_code, resolution);
CREATE INDEX idx_receiving_divergences_notice ON receiving_divergences(notice_id) WHERE notice_id IS NOT NULL;

-- ---- Supplier EDI (structured) (FEDS) ----
CREATE TABLE supplier_edi_messages (
  id BIGSERIAL PRIMARY KEY,
  enterprise_code BIGINT NOT NULL DEFAULT 1,
  supplier_code BIGINT,
  direction TEXT NOT NULL,                    -- INBOUND|OUTBOUND
  message_type TEXT NOT NULL,                 -- ORDER_CONFIRMATION|SHIP_NOTICE|INVOICE|ORDER|OTHER
  purchase_order_code BIGINT,
  external_reference TEXT,
  status TEXT NOT NULL DEFAULT 'RECEIVED',    -- RECEIVED|PROCESSED|WITH_DIVERGENCE|ERROR|SENT
  divergence_count INTEGER NOT NULL DEFAULT 0,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  notes TEXT,
  created_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  processed_at TIMESTAMPTZ,
  CHECK (direction IN ('INBOUND','OUTBOUND')),
  CHECK (status IN ('RECEIVED','PROCESSED','WITH_DIVERGENCE','ERROR','SENT'))
);
CREATE INDEX idx_supplier_edi_messages_supplier ON supplier_edi_messages(supplier_code, created_at DESC);
CREATE INDEX idx_supplier_edi_messages_po ON supplier_edi_messages(purchase_order_code) WHERE purchase_order_code IS NOT NULL;

CREATE TABLE supplier_edi_lines (
  id BIGSERIAL PRIMARY KEY,
  message_id BIGINT NOT NULL REFERENCES supplier_edi_messages(id) ON DELETE CASCADE,
  purchase_order_item_code BIGINT,
  item_code BIGINT,
  mask TEXT NOT NULL DEFAULT '',
  confirmed_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  confirmed_price NUMERIC(15,4) NOT NULL DEFAULT 0,
  confirmed_date DATE,
  divergence TEXT,                            -- comma list of QTY/PRICE/DATE, null when OK
  notes TEXT
);
CREATE INDEX idx_supplier_edi_lines_message ON supplier_edi_lines(message_id);

-- ---- Import landed cost (FREC0203 / FIMP) ----
CREATE TABLE import_processes (
  id BIGSERIAL PRIMARY KEY,
  enterprise_code BIGINT NOT NULL DEFAULT 1,
  process_number BIGINT GENERATED ALWAYS AS IDENTITY,
  supplier_code BIGINT,
  purchase_order_code BIGINT,
  reference TEXT,                             -- DI/DUIMP number
  incoterm TEXT,
  currency TEXT NOT NULL DEFAULT 'USD',
  exchange_rate NUMERIC(15,6) NOT NULL DEFAULT 1,
  apportion_basis TEXT NOT NULL DEFAULT 'VALUE',  -- VALUE|WEIGHT|QUANTITY
  status TEXT NOT NULL DEFAULT 'OPEN',        -- OPEN|NATIONALIZED|CANCELLED
  notes TEXT,
  created_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  nationalized_at TIMESTAMPTZ,
  CHECK (exchange_rate > 0),
  CHECK (apportion_basis IN ('VALUE','WEIGHT','QUANTITY')),
  CHECK (status IN ('OPEN','NATIONALIZED','CANCELLED'))
);
CREATE INDEX idx_import_processes_status ON import_processes(status, created_at DESC);

CREATE TABLE import_process_items (
  id BIGSERIAL PRIMARY KEY,
  process_id BIGINT NOT NULL REFERENCES import_processes(id) ON DELETE CASCADE,
  item_code BIGINT NOT NULL,
  mask TEXT NOT NULL DEFAULT '',
  quantity NUMERIC(15,4) NOT NULL DEFAULT 0,
  weight NUMERIC(15,4) NOT NULL DEFAULT 0,
  fob_unit_price NUMERIC(15,6) NOT NULL DEFAULT 0,        -- foreign currency
  apportioned_expenses NUMERIC(15,4) NOT NULL DEFAULT 0,  -- local currency (computed)
  landed_unit_cost NUMERIC(15,6) NOT NULL DEFAULT 0,      -- local currency (computed)
  notes TEXT,
  CHECK (quantity >= 0),
  CHECK (weight >= 0)
);
CREATE INDEX idx_import_process_items_process ON import_process_items(process_id);

CREATE TABLE import_expenses (
  id BIGSERIAL PRIMARY KEY,
  process_id BIGINT NOT NULL REFERENCES import_processes(id) ON DELETE CASCADE,
  expense_type TEXT NOT NULL,                 -- FREIGHT|INSURANCE|II|IPI|ICMS|PIS|COFINS|SISCOMEX|STORAGE|OTHER
  amount NUMERIC(15,4) NOT NULL DEFAULT 0,    -- local currency
  in_item_cost BOOLEAN NOT NULL DEFAULT TRUE, -- whether it composes landed cost
  notes TEXT,
  CHECK (amount >= 0)
);
CREATE INDEX idx_import_expenses_process ON import_expenses(process_id);

-- ---- Procurement parameters panel (FUTL0125 *) ----
CREATE TABLE procurement_parameters (
  id BIGSERIAL PRIMARY KEY,
  enterprise_code BIGINT NOT NULL DEFAULT 1,
  domain TEXT NOT NULL,                       -- PURCHASE_TABLE|PURCHASE_ORDER|QUOTATION|REQUISITION|RECEIVING_NOTICE|INSPECTION|SUPPLIER_EVALUATION|CONTRACT|SUPPLIER|NF_ENTRY
  param_key TEXT NOT NULL,
  param_value TEXT NOT NULL DEFAULT '',
  value_type TEXT NOT NULL DEFAULT 'STRING',  -- STRING|NUMBER|BOOL|JSON
  description TEXT,
  updated_by UUID,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (enterprise_code, domain, param_key),
  CHECK (value_type IN ('STRING','NUMBER','BOOL','JSON'))
);
CREATE INDEX idx_procurement_parameters_domain ON procurement_parameters(enterprise_code, domain);

-- ---- Supplier homologation (FAVF0203) ----
CREATE TABLE supplier_homologations (
  id BIGSERIAL PRIMARY KEY,
  supplier_code BIGINT NOT NULL,
  status TEXT NOT NULL DEFAULT 'PENDING',     -- HOMOLOGATED|CONDITIONAL|PENDING|SUSPENDED|REJECTED
  iqf_score NUMERIC(7,4),
  category TEXT,
  valid_until DATE,
  notes TEXT,
  decided_by UUID,
  decided_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (status IN ('HOMOLOGATED','CONDITIONAL','PENDING','SUSPENDED','REJECTED'))
);
CREATE INDEX idx_supplier_homologations_supplier ON supplier_homologations(supplier_code, decided_at DESC);
