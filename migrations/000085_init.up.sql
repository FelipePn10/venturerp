BEGIN;

ALTER TABLE overhead_allocations
    ADD COLUMN IF NOT EXISTS cost_center_code INTEGER;

ALTER TABLE overhead_allocations
    ADD COLUMN IF NOT EXISTS plan_account_code INTEGER;

ALTER TABLE overhead_allocations
    ADD COLUMN IF NOT EXISTS base_code BIGINT;

-- Garantir UNIQUE no code (idempotente: a constraint já é criada na migration 084)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_overhead_allocations_code') THEN
        ALTER TABLE overhead_allocations
            ADD CONSTRAINT uq_overhead_allocations_code UNIQUE (code);
    END IF;
END $$;

-- Popular cost_center_code
UPDATE overhead_allocations oa
SET cost_center_code = cc.code
    FROM cost_centers cc
WHERE oa.cost_center_id = cc.id
  AND oa.cost_center_code IS NULL;

-- FK
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_overhead_allocations_cost_center_code') THEN
        ALTER TABLE overhead_allocations
            ADD CONSTRAINT fk_overhead_allocations_cost_center_code
                FOREIGN KEY (cost_center_code) REFERENCES cost_centers(code);
    END IF;
END $$;

-- Índices
CREATE INDEX IF NOT EXISTS idx_overhead_allocations_cost_center_code
    ON overhead_allocations(cost_center_code);

CREATE INDEX IF NOT EXISTS idx_overhead_allocations_code
    ON overhead_allocations(code);

-- =====================================================
-- overhead_allocation_targets
-- =====================================================

ALTER TABLE overhead_allocation_targets
    ADD COLUMN IF NOT EXISTS overhead_code BIGINT;

ALTER TABLE overhead_allocation_targets
    ADD COLUMN IF NOT EXISTS cost_center_code INTEGER;

-- Popular overhead_code
UPDATE overhead_allocation_targets oat
SET overhead_code = oa.code
    FROM overhead_allocations oa
WHERE oat.overhead_id = oa.id
  AND oat.overhead_code IS NULL;

-- Popular cost_center_code
UPDATE overhead_allocation_targets oat
SET cost_center_code = cc.code
    FROM cost_centers cc
WHERE oat.cost_center_id = cc.id
  AND oat.cost_center_code IS NULL;

-- FKs
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_oat_overhead_code') THEN
        ALTER TABLE overhead_allocation_targets
            ADD CONSTRAINT fk_oat_overhead_code
                FOREIGN KEY (overhead_code) REFERENCES overhead_allocations(code);
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_oat_cost_center_code') THEN
        ALTER TABLE overhead_allocation_targets
            ADD CONSTRAINT fk_oat_cost_center_code
                FOREIGN KEY (cost_center_code) REFERENCES cost_centers(code);
    END IF;
END $$;

-- Índices
CREATE INDEX IF NOT EXISTS idx_oat_overhead_code
    ON overhead_allocation_targets(overhead_code);

CREATE INDEX IF NOT EXISTS idx_oat_cost_center_code
    ON overhead_allocation_targets(cost_center_code);

COMMIT;
