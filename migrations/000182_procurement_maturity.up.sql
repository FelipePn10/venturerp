CREATE TYPE procurement_record_type AS ENUM (
  'RECEIVING_INSPECTION',
  'RECEIVING_NOTICE',
  'SUPPLIER_EVALUATION',
  'APPROVAL_LIMIT',
  'SUPPLIER_CONTRACT',
  'RECEIVING_CHECKLIST',
  'RECEIVING_LABEL',
  'SUPPLIER_EDI',
  'IMPORT_PROCESS'
);

CREATE TYPE procurement_record_status AS ENUM (
  'DRAFT',
  'OPEN',
  'IN_REVIEW',
  'APPROVED',
  'REJECTED',
  'PARTIAL',
  'CLOSED',
  'CANCELLED'
);

CREATE TABLE procurement_records (
  id BIGSERIAL PRIMARY KEY,
  record_type procurement_record_type NOT NULL,
  status procurement_record_status NOT NULL DEFAULT 'OPEN',
  supplier_code BIGINT,
  purchase_order_code BIGINT,
  purchase_order_item_code BIGINT,
  item_code BIGINT,
  mask TEXT NOT NULL DEFAULT '',
  warehouse_id BIGINT,
  quantity NUMERIC(15,4) NOT NULL DEFAULT 0,
  reference TEXT,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  closed_at TIMESTAMPTZ,
  created_by UUID,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (quantity >= 0)
);

CREATE INDEX idx_procurement_records_type_status ON procurement_records(record_type, status);
CREATE INDEX idx_procurement_records_supplier ON procurement_records(supplier_code) WHERE supplier_code IS NOT NULL;
CREATE INDEX idx_procurement_records_po ON procurement_records(purchase_order_code) WHERE purchase_order_code IS NOT NULL;
CREATE INDEX idx_procurement_records_item ON procurement_records(item_code) WHERE item_code IS NOT NULL;

CREATE TABLE procurement_inspection_dispositions (
  id BIGSERIAL PRIMARY KEY,
  record_id BIGINT NOT NULL REFERENCES procurement_records(id) ON DELETE CASCADE,
  approved_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  rejected_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  quarantine_warehouse_id BIGINT,
  destination_warehouse_id BIGINT,
  reason TEXT,
  disposed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  disposed_by UUID,
  CHECK (approved_qty >= 0),
  CHECK (rejected_qty >= 0)
);

CREATE TABLE supplier_scorecard_snapshots (
  id BIGSERIAL PRIMARY KEY,
  supplier_code BIGINT NOT NULL,
  period_start DATE NOT NULL,
  period_end DATE NOT NULL,
  quality_score NUMERIC(7,4) NOT NULL DEFAULT 100,
  delivery_score NUMERIC(7,4) NOT NULL DEFAULT 100,
  commercial_score NUMERIC(7,4) NOT NULL DEFAULT 100,
  service_score NUMERIC(7,4) NOT NULL DEFAULT 100,
  overall_score NUMERIC(7,4) NOT NULL DEFAULT 100,
  total_receipts INTEGER NOT NULL DEFAULT 0,
  rejected_receipts INTEGER NOT NULL DEFAULT 0,
  late_receipts INTEGER NOT NULL DEFAULT 0,
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by UUID,
  CHECK (period_end >= period_start),
  UNIQUE (supplier_code, period_start, period_end)
);
