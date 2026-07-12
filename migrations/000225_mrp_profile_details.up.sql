CREATE TABLE mrp_profile_details (
    id BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    plan_code BIGINT NOT NULL,
    item_code BIGINT NOT NULL,
    need_date DATE NOT NULL,
    detail_type VARCHAR(30) NOT NULL,
    source_code BIGINT,
    parent_item_code BIGINT,
    quantity NUMERIC(18,6) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mrp_profile_details_tenant_plan
    ON mrp_profile_details (enterprise_id, plan_code, item_code, need_date);
