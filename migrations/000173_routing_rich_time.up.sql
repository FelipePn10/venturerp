BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Rich time model for the manufacturing route (enterprise+).
--
-- The route becomes the single source of truth for operation times, replacing the
-- flat standard_time/setup_time pair with separately measured components:
--   • setup   — one-off per lot (already existed)
--   • run     — machine/processing time per `run_time_base_qty` pieces
--   • labor   — direct-labor time per `run_time_base_qty` pieces (0 ⇒ equals run)
--   • queue   — waiting-in-queue time before the operation (fixed per lot)
--   • wait    — post-processing wait/cure time (fixed per lot)
--   • move    — transport time to the next operation (fixed per lot)
--
-- crew_size multiplies labor for costing. time_unit lets each op be measured in
-- MIN/HORA/DIA. Legacy standard_time/setup_time columns are kept and back-filled so
-- existing sqlc-generated code and data keep working during the transition.
-- ─────────────────────────────────────────────────────────────────────────────

-- ─── operations (catalog defaults) ─────────────────────────────────────────────

ALTER TABLE operations
    ADD COLUMN IF NOT EXISTS run_time          NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS labor_time        NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS run_time_base_qty NUMERIC(15,4) NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS queue_time        NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS wait_time         NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS move_time         NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS crew_size         NUMERIC(6,2)  NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS time_unit         VARCHAR(10)   NOT NULL DEFAULT 'HORA';

ALTER TABLE operations
    ADD CONSTRAINT chk_operations_time_unit CHECK (time_unit IN ('MIN', 'HORA', 'DIA'));
ALTER TABLE operations
    ADD CONSTRAINT chk_operations_base_qty CHECK (run_time_base_qty > 0);
ALTER TABLE operations
    ADD CONSTRAINT chk_operations_crew CHECK (crew_size > 0);

-- Back-fill: the legacy flat standard_time becomes the run (machine) time.
UPDATE operations SET run_time = standard_time WHERE run_time = 0 AND standard_time > 0;

-- ─── route_operations (nullable overrides — NULL = inherit from operation) ──────

ALTER TABLE route_operations
    ADD COLUMN IF NOT EXISTS run_time          NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS labor_time        NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS run_time_base_qty NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS queue_time        NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS wait_time         NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS move_time         NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS crew_size         NUMERIC(6,2),
    ADD COLUMN IF NOT EXISTS time_unit         VARCHAR(10);

ALTER TABLE route_operations
    ADD CONSTRAINT chk_route_ops_time_unit CHECK (time_unit IS NULL OR time_unit IN ('MIN', 'HORA', 'DIA'));
ALTER TABLE route_operations
    ADD CONSTRAINT chk_route_ops_base_qty CHECK (run_time_base_qty IS NULL OR run_time_base_qty > 0);
ALTER TABLE route_operations
    ADD CONSTRAINT chk_route_ops_crew CHECK (crew_size IS NULL OR crew_size > 0);

-- Back-fill: a route op that already overrode standard_time keeps that as its run override.
UPDATE route_operations SET run_time = standard_time WHERE run_time IS NULL AND standard_time IS NOT NULL;

COMMIT;
