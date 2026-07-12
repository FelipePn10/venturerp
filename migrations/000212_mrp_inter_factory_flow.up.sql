ALTER TABLE mrp_planned_suggestions
    ADD COLUMN IF NOT EXISTS inter_factory BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS source_enterprise_code BIGINT,
    ADD COLUMN IF NOT EXISTS auto_release BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE planned_orders
    ADD COLUMN IF NOT EXISTS inter_factory BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS source_enterprise_code BIGINT,
    ADD COLUMN IF NOT EXISTS auto_release BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_mrp_suggestions_inter_factory
    ON mrp_planned_suggestions (enterprise_id, source_enterprise_code, auto_release)
    WHERE inter_factory = TRUE;
