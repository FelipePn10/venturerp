ALTER TABLE users
    ADD COLUMN auth_version BIGINT NOT NULL DEFAULT 1;

CREATE TABLE password_change_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    requested_by UUID NOT NULL REFERENCES users(id),
    approved_by UUID REFERENCES users(id),
    rejected_by UUID REFERENCES users(id),
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED', 'USED', 'EXPIRED')),
    expires_at TIMESTAMPTZ,
    approved_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    used_at TIMESTAMPTZ,
    rejection_reason VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (requested_by = user_id),
    CHECK (approved_by IS NULL OR approved_by <> user_id)
);

CREATE UNIQUE INDEX password_change_requests_one_active_user_idx
    ON password_change_requests (enterprise_id, user_id)
    WHERE status IN ('PENDING', 'APPROVED');

CREATE INDEX password_change_requests_tenant_status_idx
    ON password_change_requests (enterprise_id, status, created_at DESC);
