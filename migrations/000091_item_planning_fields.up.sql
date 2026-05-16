BEGIN;

-- Add planning extension fields to items via a new table
CREATE TABLE IF NOT EXISTS item_planning_extras (
    id              BIGSERIAL PRIMARY KEY,
    item_code       BIGINT NOT NULL UNIQUE,
    safety_time     INT NOT NULL DEFAULT 0,
    coverage        INT NOT NULL DEFAULT 0,
    grouping_key    VARCHAR(50),
    is_critical     BOOLEAN NOT NULL DEFAULT FALSE,
    maximum_stock   NUMERIC(15,4) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_item_planning_extras_item ON item_planning_extras(item_code);

-- Add use_tank_date to items for param 12
ALTER TABLE item_planning_extras
  ADD COLUMN IF NOT EXISTS use_tank_date BOOLEAN NOT NULL DEFAULT FALSE;

COMMIT;
