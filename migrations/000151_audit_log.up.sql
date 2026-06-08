-- Audit trail: an append-only record of every authenticated mutating request
-- (POST/PUT/PATCH/DELETE) — "who changed what, when". Captured at the HTTP layer
-- by the Audit middleware, written asynchronously so it never blocks a request.

CREATE TABLE IF NOT EXISTS public.audit_log (
    id          BIGSERIAL    PRIMARY KEY,
    occurred_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    request_id  VARCHAR(64),               -- correlation ID (X-Request-ID)
    user_id     VARCHAR(64),               -- actor (from JWT); NULL if unauthenticated
    user_role   VARCHAR(32),               -- actor role at the time of the action
    method      VARCHAR(10)  NOT NULL,      -- HTTP verb
    route       VARCHAR(255) NOT NULL,      -- chi route pattern, e.g. /api/sales-order/{code}
    path        VARCHAR(512) NOT NULL,      -- concrete path requested
    query       VARCHAR(1024),             -- raw query string, if any
    status      INTEGER      NOT NULL,      -- response status code
    ip          VARCHAR(64),               -- client IP
    user_agent  VARCHAR(512),
    latency_ms  BIGINT
);

CREATE INDEX IF NOT EXISTS idx_audit_log_occurred_at ON public.audit_log (occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_user_id     ON public.audit_log (user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_route       ON public.audit_log (route);
