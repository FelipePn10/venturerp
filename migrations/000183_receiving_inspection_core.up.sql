CREATE TYPE receiving_inspection_basis AS ENUM ('ITEM', 'CLASSIFICATION');
CREATE TYPE receiving_inspection_step_kind AS ENUM ('VALUE', 'ATTRIBUTE', 'STRUCTURE');
CREATE TYPE receiving_inspection_appointment_mode AS ENUM (
  'ALL_MEASUREMENTS',
  'SINGLE_INTERVAL',
  'MULTIPLE_INTERVAL',
  'STATUS_ONLY'
);
CREATE TYPE receiving_inspection_order_source AS ENUM (
  'PURCHASE_RECEIPT',
  'RECEIVING_NOTICE',
  'FISCAL_ENTRY',
  'MANUAL'
);
CREATE TYPE receiving_inspection_order_status AS ENUM (
  'PENDING_INSPECTION',
  'PENDING_ANALYSIS',
  'APPROVED',
  'REJECTED',
  'PARTIAL',
  'CANCELLED',
  'SKIPPED'
);
CREATE TYPE receiving_inspection_treatment AS ENUM (
  'ACCEPT_WITH_RESTRICTION',
  'RETURN_TO_SUPPLIER',
  'SCRAP',
  'REWORK',
  'SORTING',
  'CONCESSION'
);

CREATE TABLE receiving_inspection_routes (
  id BIGSERIAL PRIMARY KEY,
  enterprise_code BIGINT NOT NULL DEFAULT 1,
  basis receiving_inspection_basis NOT NULL,
  item_code BIGINT,
  classification_code TEXT,
  mask TEXT NOT NULL DEFAULT '',
  inspection_warehouse_id BIGINT NOT NULL,
  handling_type TEXT,
  storage_type TEXT,
  route_type TEXT,
  market_type TEXT,
  inspection_type TEXT,
  valid_from DATE NOT NULL DEFAULT CURRENT_DATE,
  valid_to DATE,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by UUID,
  CHECK (valid_to IS NULL OR valid_to >= valid_from),
  CHECK (
    (basis = 'ITEM' AND item_code IS NOT NULL)
    OR (basis = 'CLASSIFICATION' AND classification_code IS NOT NULL AND classification_code <> '')
  )
);

CREATE INDEX idx_receiving_inspection_routes_item ON receiving_inspection_routes(enterprise_code, item_code, mask, valid_from DESC)
  WHERE basis = 'ITEM' AND is_active = TRUE;
CREATE INDEX idx_receiving_inspection_routes_classification ON receiving_inspection_routes(enterprise_code, classification_code, valid_from DESC)
  WHERE basis = 'CLASSIFICATION' AND is_active = TRUE;

CREATE TABLE receiving_inspection_route_steps (
  id BIGSERIAL PRIMARY KEY,
  route_id BIGINT NOT NULL REFERENCES receiving_inspection_routes(id) ON DELETE CASCADE,
  sequence INTEGER NOT NULL,
  inspection_name TEXT NOT NULL,
  kind receiving_inspection_step_kind NOT NULL,
  appointment_mode receiving_inspection_appointment_mode NOT NULL,
  is_required BOOLEAN NOT NULL DEFAULT TRUE,
  emits_label BOOLEAN NOT NULL DEFAULT FALSE,
  instrument_group TEXT,
  sample_type TEXT,
  sample_unit TEXT,
  sample_qty NUMERIC(15,4) NOT NULL DEFAULT 1,
  acceptance_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  rejection_qty NUMERIC(15,4) NOT NULL DEFAULT 1,
  norm TEXT,
  reference TEXT,
  valid_to DATE,
  nominal_value NUMERIC(15,6),
  min_value NUMERIC(15,6),
  max_value NUMERIC(15,6),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (sample_qty > 0),
  CHECK (acceptance_qty >= 0),
  CHECK (rejection_qty >= 0),
  CHECK (min_value IS NULL OR max_value IS NULL OR min_value <= max_value),
  UNIQUE (route_id, sequence)
);

CREATE TABLE receiving_inspection_step_attributes (
  id BIGSERIAL PRIMARY KEY,
  step_id BIGINT NOT NULL REFERENCES receiving_inspection_route_steps(id) ON DELETE CASCADE,
  description TEXT NOT NULL,
  is_approved BOOLEAN NOT NULL,
  UNIQUE (step_id, description)
);

CREATE TABLE receiving_inspection_orders (
  id BIGSERIAL PRIMARY KEY,
  order_number BIGINT GENERATED ALWAYS AS IDENTITY,
  route_id BIGINT REFERENCES receiving_inspection_routes(id),
  procurement_record_id BIGINT REFERENCES procurement_records(id),
  source receiving_inspection_order_source NOT NULL,
  supplier_code BIGINT,
  purchase_order_code BIGINT,
  purchase_order_item_code BIGINT,
  fiscal_entry_code BIGINT,
  receiving_notice_code BIGINT,
  item_code BIGINT NOT NULL,
  mask TEXT NOT NULL DEFAULT '',
  lot TEXT,
  serial_number TEXT,
  warehouse_id BIGINT NOT NULL,
  quantity NUMERIC(15,4) NOT NULL,
  inspected_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  approved_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  rejected_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  rework_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  restricted_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  status receiving_inspection_order_status NOT NULL DEFAULT 'PENDING_INSPECTION',
  certificate TEXT,
  supplier_note TEXT,
  model TEXT,
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by UUID,
  CHECK (quantity > 0),
  CHECK (inspected_qty >= 0),
  CHECK (approved_qty >= 0),
  CHECK (rejected_qty >= 0),
  CHECK (rework_qty >= 0),
  CHECK (restricted_qty >= 0)
);

CREATE INDEX idx_receiving_inspection_orders_status ON receiving_inspection_orders(status, created_at DESC);
CREATE INDEX idx_receiving_inspection_orders_item ON receiving_inspection_orders(item_code, mask, created_at DESC);
CREATE INDEX idx_receiving_inspection_orders_supplier ON receiving_inspection_orders(supplier_code, created_at DESC) WHERE supplier_code IS NOT NULL;

CREATE TABLE receiving_inspection_results (
  id BIGSERIAL PRIMARY KEY,
  order_id BIGINT NOT NULL REFERENCES receiving_inspection_orders(id) ON DELETE CASCADE,
  step_id BIGINT REFERENCES receiving_inspection_route_steps(id),
  sequence INTEGER NOT NULL,
  sample_index INTEGER NOT NULL DEFAULT 1,
  measured_value NUMERIC(15,6),
  min_value NUMERIC(15,6),
  max_value NUMERIC(15,6),
  attribute_description TEXT,
  is_approved BOOLEAN NOT NULL,
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by UUID,
  CHECK (sample_index > 0)
);

CREATE INDEX idx_receiving_inspection_results_order ON receiving_inspection_results(order_id, sequence, sample_index);

CREATE TABLE receiving_inspection_analyses (
  id BIGSERIAL PRIMARY KEY,
  order_id BIGINT NOT NULL UNIQUE REFERENCES receiving_inspection_orders(id) ON DELETE CASCADE,
  conform_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  rejected_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  rework_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  restricted_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  treatment receiving_inspection_treatment NOT NULL,
  affects_supplier_score BOOLEAN NOT NULL DEFAULT TRUE,
  notes TEXT,
  analyzed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  analyzed_by UUID,
  CHECK (conform_qty >= 0),
  CHECK (rejected_qty >= 0),
  CHECK (rework_qty >= 0),
  CHECK (restricted_qty >= 0)
);
