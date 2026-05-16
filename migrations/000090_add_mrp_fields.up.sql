BEGIN;

-- Add MaximumStock to items for MIN_MAX planning (via stock_snapshots extension)
ALTER TABLE stock_snapshots
  ADD COLUMN IF NOT EXISTS maximum_stock NUMERIC(15,4) NOT NULL DEFAULT 0;

-- Add planning fields to item warehouse data (via stock_snapshots extension)
ALTER TABLE stock_snapshots
  ADD COLUMN IF NOT EXISTS min_max_active BOOLEAN NOT NULL DEFAULT FALSE;

-- Add safety_time to items for planning
-- (Store in the planned_orders table as an override field)
ALTER TABLE planned_orders
  ADD COLUMN IF NOT EXISTS safety_time_days INT NOT NULL DEFAULT 0;

-- Add coverage field for items (days of coverage desired)
ALTER TABLE planned_orders
  ADD COLUMN IF NOT EXISTS coverage_days INT NOT NULL DEFAULT 0;

-- Add kanban table
CREATE TABLE IF NOT EXISTS kanban_cards (
    id              BIGSERIAL PRIMARY KEY,
    item_code       BIGINT NOT NULL,
    card_count      INT NOT NULL DEFAULT 1,
    quantity_per_card NUMERIC(15,4) NOT NULL,
    reorder_point   NUMERIC(15,4) NOT NULL,
    container_type  VARCHAR(50),
    status          VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_kanban_cards_item ON kanban_cards(item_code);

-- Add MPS (Master Production Schedule) table
CREATE TABLE IF NOT EXISTS mps_schedule (
    id              BIGSERIAL PRIMARY KEY,
    item_code       BIGINT NOT NULL,
    mask            VARCHAR(200) NOT NULL DEFAULT '',
    period_type     VARCHAR(10) NOT NULL DEFAULT 'MONTH',
    period_value    INT NOT NULL,
    year            INT NOT NULL,
    quantity        NUMERIC(15,4) NOT NULL,
    is_firm         BOOLEAN NOT NULL DEFAULT FALSE,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID NOT NULL,
    UNIQUE(item_code, mask, period_type, period_value, year)
);

CREATE INDEX IF NOT EXISTS idx_mps_schedule_item ON mps_schedule(item_code);
CREATE INDEX IF NOT EXISTS idx_mps_schedule_period ON mps_schedule(year, period_type, period_value);

COMMIT;
