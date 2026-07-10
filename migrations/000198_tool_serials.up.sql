BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Ficha de Produção da Ferramenta (Tool Production Sheet).
--
-- A tool (`tools`) is a master record (a die, jig, fixture or cutting tool). In
-- the shop floor a single tool master usually has several *physical copies*,
-- each identified by its own serial number and worn independently. `tool_serials`
-- models those physical instances.
--
-- The production sheet binds, per production-order operation, which physical
-- serial of the required tool will actually run the job:
--   * `production_order_operation_tool_serials` — the active binding (1 serial per
--     operation × tool).
--   * `tool_serial_substitutions` — an audit trail of every replacement, so the
--     shop floor keeps full traceability of which serial ran which operation.
-- ─────────────────────────────────────────────────────────────────────────────

-- Physical instances (serial numbers) of a tool master.
CREATE TABLE IF NOT EXISTS tool_serials (
    id            BIGSERIAL PRIMARY KEY,
    tool_id       BIGINT NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    serial_number VARCHAR(60) NOT NULL,
    status        VARCHAR(12) NOT NULL DEFAULT 'ATIVA'
                  CHECK (status IN ('ATIVA', 'MANUTENCAO', 'INATIVA', 'BAIXADA')),
    life_used     NUMERIC(15,4) NOT NULL DEFAULT 0,   -- per-serial life consumption
    location      VARCHAR(120),
    notes         TEXT,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by    UUID NOT NULL,
    UNIQUE (tool_id, serial_number)
);

CREATE INDEX IF NOT EXISTS idx_tool_serials_tool ON tool_serials(tool_id);

-- Active binding of a physical serial to a production-order operation. One serial
-- per (operation, tool): re-assigning the same pair updates the binding in place.
CREATE TABLE IF NOT EXISTS production_order_operation_tool_serials (
    id                            BIGSERIAL PRIMARY KEY,
    production_order_operation_id BIGINT NOT NULL REFERENCES production_order_operations(id) ON DELETE CASCADE,
    tool_id                       BIGINT NOT NULL REFERENCES tools(id),
    tool_serial_id                BIGINT NOT NULL REFERENCES tool_serials(id),
    assigned_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by                   UUID NOT NULL,
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (production_order_operation_id, tool_id)
);

CREATE INDEX IF NOT EXISTS idx_poots_op ON production_order_operation_tool_serials(production_order_operation_id);
CREATE INDEX IF NOT EXISTS idx_poots_serial ON production_order_operation_tool_serials(tool_serial_id);

-- Audit trail of serial replacements on an operation.
CREATE TABLE IF NOT EXISTS tool_serial_substitutions (
    id                            BIGSERIAL PRIMARY KEY,
    production_order_operation_id BIGINT NOT NULL REFERENCES production_order_operations(id) ON DELETE CASCADE,
    tool_id                       BIGINT NOT NULL REFERENCES tools(id),
    old_serial_id                 BIGINT REFERENCES tool_serials(id),
    new_serial_id                 BIGINT NOT NULL REFERENCES tool_serials(id),
    reason                        TEXT,
    substituted_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    substituted_by                UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tss_op ON tool_serial_substitutions(production_order_operation_id);

COMMIT;
