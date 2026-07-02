BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Alternative resources per route operation (enterprise+ R5).
--
-- A route operation names a PRIMARY work center (route_operations.work_center_id,
-- resolved as effective_work_center_id). This table records ALTERNATIVE work centers
-- that can also run the operation, each with a priority (1 = most preferred) and a
-- time_factor that scales the operation's time on that resource (1.0 = same as base,
-- 1.2 = 20% slower, 0.9 = 10% faster). The APS/CRP can pick an alternative when the
-- primary is overloaded; for now the data is exposed and the primary drives costing.
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS route_operation_resources (
    id                 BIGSERIAL PRIMARY KEY,
    route_operation_id BIGINT NOT NULL REFERENCES route_operations(id) ON DELETE CASCADE,
    work_center_id     BIGINT NOT NULL REFERENCES machine_types(id),
    priority           SMALLINT NOT NULL DEFAULT 1,
    time_factor        NUMERIC(8,4) NOT NULL DEFAULT 1 CHECK (time_factor > 0),
    is_primary         BOOLEAN NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (route_operation_id, work_center_id)
);

CREATE INDEX IF NOT EXISTS idx_route_op_resources_op ON route_operation_resources(route_operation_id);
CREATE INDEX IF NOT EXISTS idx_route_op_resources_wc ON route_operation_resources(work_center_id);

-- At most one primary resource per operation.
CREATE UNIQUE INDEX IF NOT EXISTS idx_route_op_resources_primary
    ON route_operation_resources(route_operation_id) WHERE is_primary;

COMMIT;
