CREATE TABLE IF NOT EXISTS production_plan_inter_factories (
    id BIGSERIAL PRIMARY KEY,
    plan_code BIGINT NOT NULL REFERENCES production_plans(code) ON DELETE CASCADE,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    source_enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    auto_release BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_plan_inter_factory_not_self CHECK (enterprise_id <> source_enterprise_id),
    CONSTRAINT uq_plan_inter_factory UNIQUE (enterprise_id, plan_code, source_enterprise_id)
);

CREATE INDEX IF NOT EXISTS idx_plan_inter_factories_plan
    ON production_plan_inter_factories (enterprise_id, plan_code);
CREATE INDEX IF NOT EXISTS idx_plan_inter_factories_source
    ON production_plan_inter_factories (source_enterprise_id, auto_release);
