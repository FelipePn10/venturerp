BEGIN;

CREATE TABLE restriction_reasons (
    id          BIGSERIAL PRIMARY KEY,
    code        BIGSERIAL UNIQUE NOT NULL,
    description TEXT NOT NULL,
    situation   VARCHAR(10) NOT NULL DEFAULT 'ACTIVE',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE restrictions
    ADD COLUMN IF NOT EXISTS customer_code BIGINT;

ALTER TABLE restrictions
    DROP CONSTRAINT IF EXISTS fk_restrictions_reason;

ALTER TABLE restrictions
    ADD CONSTRAINT fk_restrictions_reason
        FOREIGN KEY (reason_code) REFERENCES restriction_reasons(id)
        ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_restrictions_customer ON restrictions(customer_code);

COMMIT;
