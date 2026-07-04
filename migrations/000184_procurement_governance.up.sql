-- Purchase approval limits (alçada de valores) — normalized rule + enforcement.
CREATE TABLE purchase_approval_limits (
  id BIGSERIAL PRIMARY KEY,
  enterprise_code BIGINT NOT NULL DEFAULT 1,
  scope TEXT NOT NULL DEFAULT 'GLOBAL',   -- GLOBAL | SUPPLIER | COST_CENTER | CATEGORY
  scope_ref TEXT,                         -- supplier code / cost center / category when scope <> GLOBAL
  currency TEXT NOT NULL DEFAULT 'BRL',
  auto_approve_max NUMERIC(15,2) NOT NULL DEFAULT 0,  -- total <= this is auto-approved
  block_above NUMERIC(15,2),                          -- total > this is hard-blocked even for authorizers
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  valid_from DATE NOT NULL DEFAULT CURRENT_DATE,
  valid_to DATE,
  notes TEXT,
  created_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (auto_approve_max >= 0),
  CHECK (block_above IS NULL OR block_above >= auto_approve_max),
  CHECK (valid_to IS NULL OR valid_to >= valid_from),
  CHECK (scope IN ('GLOBAL','SUPPLIER','COST_CENTER','CATEGORY')),
  CHECK (scope = 'GLOBAL' OR (scope_ref IS NOT NULL AND scope_ref <> ''))
);

CREATE INDEX idx_purchase_approval_limits_lookup
  ON purchase_approval_limits(enterprise_code, scope, scope_ref, valid_from DESC)
  WHERE is_active;

-- Supplier contracts (contrato de fornecedores) — header + normalized item lines.
CREATE TABLE supplier_contracts (
  id BIGSERIAL PRIMARY KEY,
  enterprise_code BIGINT NOT NULL DEFAULT 1,
  supplier_code BIGINT NOT NULL,
  contract_number TEXT NOT NULL,
  description TEXT,
  status TEXT NOT NULL DEFAULT 'DRAFT',   -- DRAFT | ACTIVE | SUSPENDED | CLOSED | CANCELLED
  currency TEXT NOT NULL DEFAULT 'BRL',
  valid_from DATE NOT NULL DEFAULT CURRENT_DATE,
  valid_to DATE,
  price_index TEXT,                        -- índice de reajuste (IGPM/IPCA/...)
  notes TEXT,
  created_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (enterprise_code, supplier_code, contract_number),
  CHECK (status IN ('DRAFT','ACTIVE','SUSPENDED','CLOSED','CANCELLED')),
  CHECK (valid_to IS NULL OR valid_to >= valid_from)
);

CREATE INDEX idx_supplier_contracts_supplier ON supplier_contracts(supplier_code, status);

CREATE TABLE supplier_contract_items (
  id BIGSERIAL PRIMARY KEY,
  contract_id BIGINT NOT NULL REFERENCES supplier_contracts(id) ON DELETE CASCADE,
  item_code BIGINT NOT NULL,
  mask TEXT NOT NULL DEFAULT '',
  unit TEXT,
  contracted_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  consumed_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  unit_price NUMERIC(15,6) NOT NULL DEFAULT 0,
  min_order_qty NUMERIC(15,4) NOT NULL DEFAULT 0,
  notes TEXT,
  CHECK (contracted_qty >= 0),
  CHECK (consumed_qty >= 0),
  CHECK (unit_price >= 0),
  UNIQUE (contract_id, item_code, mask)
);

CREATE INDEX idx_supplier_contract_items_item ON supplier_contract_items(item_code, mask);
