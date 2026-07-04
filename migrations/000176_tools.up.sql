BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Tooling with useful-life tracking (enterprise+ R3).
--
-- `tools` is the master of dies, jigs, fixtures and cutting tools. Each carries a
-- useful-life budget (GOLPES/HORAS/PECAS) consumed as production runs; when the
-- consumed life reaches the limit the tool is flagged for replacement. This is a
-- differentiator vs. Focco: tool-life is wired to the shop-floor operation apontamento.
--
-- `route_operation_tools` links the tools required to run a route operation (N:N).
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS tools (
    id         BIGSERIAL PRIMARY KEY,
    code       BIGINT NOT NULL UNIQUE,
    name       VARCHAR(120) NOT NULL,
    tool_type  VARCHAR(40) NOT NULL DEFAULT 'FERRAMENTA',
    life_type  VARCHAR(10) NOT NULL DEFAULT 'PECAS' CHECK (life_type IN ('GOLPES', 'HORAS', 'PECAS')),
    life_limit NUMERIC(15,4) NOT NULL DEFAULT 0,   -- 0 = no life tracking
    life_used  NUMERIC(15,4) NOT NULL DEFAULT 0,
    cost       NUMERIC(15,4) NOT NULL DEFAULT 0,
    status     VARCHAR(12) NOT NULL DEFAULT 'ATIVA' CHECK (status IN ('ATIVA', 'MANUTENCAO', 'INATIVA')),
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tools_code ON tools(code);

CREATE TABLE IF NOT EXISTS route_operation_tools (
    id                 BIGSERIAL PRIMARY KEY,
    route_operation_id BIGINT NOT NULL REFERENCES route_operations(id) ON DELETE CASCADE,
    tool_id            BIGINT NOT NULL REFERENCES tools(id),
    qty_required       NUMERIC(10,2) NOT NULL DEFAULT 1,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (route_operation_id, tool_id)
);

CREATE INDEX IF NOT EXISTS idx_route_op_tools_op ON route_operation_tools(route_operation_id);
CREATE INDEX IF NOT EXISTS idx_route_op_tools_tool ON route_operation_tools(tool_id);

COMMIT;
