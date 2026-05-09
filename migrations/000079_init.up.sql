CREATE TABLE IF NOT EXISTS mrp_planned_suggestions (
    code             BIGSERIAL PRIMARY KEY,
    plan_code        BIGINT NOT NULL,
    item_code        BIGINT NOT NULL,
    quantity         NUMERIC(15,4) NOT NULL,
    need_date        DATE NOT NULL,
    start_date       DATE,
    order_type       VARCHAR(20) NOT NULL,
    demand_type      VARCHAR(20) NOT NULL DEFAULT 'INDEPENDENTE',
    parent_item_code BIGINT,
    llc              INT NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mrp_suggestions_plan ON mrp_planned_suggestions(plan_code);
CREATE INDEX idx_mrp_suggestions_item ON mrp_planned_suggestions(item_code);
