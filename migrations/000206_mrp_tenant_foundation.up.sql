CREATE TABLE IF NOT EXISTS user_enterprises (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, enterprise_id)
);

-- Existing installations with a single enterprise can be associated safely.
INSERT INTO user_enterprises (user_id, enterprise_id)
SELECT u.id, e.id
FROM users u
CROSS JOIN enterprise e
WHERE (SELECT COUNT(*) FROM enterprise) = 1
ON CONFLICT DO NOTHING;

ALTER TABLE production_plans ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE independent_demands ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE planned_orders ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE mrp_item_profiles ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE mrp_calculation_logs ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE stock_snapshots ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE sales_order_demands ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE configured_item_rules ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE mrp_planned_suggestions ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE mrp_exception_messages ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE planning_params ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE order_priorities ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE item_machine_times ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE kanban_cards ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE mps_schedule ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE item_planning_extras ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE machine_schedules ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE stock_balances ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE sales_divisions ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);

UPDATE production_plans p
SET enterprise_id = ue.enterprise_id
FROM user_enterprises ue
WHERE ue.user_id = p.created_by
  AND p.enterprise_id IS NULL
  AND (SELECT COUNT(*) FROM user_enterprises x WHERE x.user_id = p.created_by) = 1;

UPDATE independent_demands d
SET enterprise_id = ue.enterprise_id
FROM user_enterprises ue
WHERE ue.user_id = d.created_by
  AND d.enterprise_id IS NULL
  AND (SELECT COUNT(*) FROM user_enterprises x WHERE x.user_id = d.created_by) = 1;

UPDATE planned_orders o SET enterprise_id = p.enterprise_id
FROM production_plans p WHERE o.plan_code = p.code AND o.enterprise_id IS NULL;
UPDATE mrp_item_profiles x SET enterprise_id = p.enterprise_id
FROM production_plans p WHERE x.plan_code = p.code AND x.enterprise_id IS NULL;
UPDATE mrp_calculation_logs x SET enterprise_id = p.enterprise_id
FROM production_plans p WHERE x.plan_code = p.code AND x.enterprise_id IS NULL;
UPDATE mrp_planned_suggestions x SET enterprise_id = p.enterprise_id
FROM production_plans p WHERE x.plan_code = p.code AND x.enterprise_id IS NULL;
UPDATE mrp_exception_messages x SET enterprise_id = p.enterprise_id
FROM production_plans p WHERE x.plan_code = p.code AND x.enterprise_id IS NULL;

UPDATE planning_params SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;
UPDATE order_priorities SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;
UPDATE item_machine_times SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;
UPDATE kanban_cards SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;
UPDATE mps_schedule SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;
UPDATE item_planning_extras SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;
UPDATE machine_schedules SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;
UPDATE stock_balances SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;
UPDATE sales_divisions SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;

CREATE INDEX IF NOT EXISTS idx_user_enterprises_enterprise ON user_enterprises (enterprise_id, user_id);
CREATE INDEX IF NOT EXISTS idx_independent_demands_tenant ON independent_demands (enterprise_id, demand_date);
CREATE INDEX IF NOT EXISTS idx_planned_orders_tenant ON planned_orders (enterprise_id, plan_code);
CREATE INDEX IF NOT EXISTS idx_mrp_profiles_tenant ON mrp_item_profiles (enterprise_id, plan_code, item_code);
CREATE INDEX IF NOT EXISTS idx_mrp_logs_tenant ON mrp_calculation_logs (enterprise_id, plan_code);
CREATE INDEX IF NOT EXISTS idx_stock_snapshots_tenant ON stock_snapshots (enterprise_id, item_code);
CREATE INDEX IF NOT EXISTS idx_sales_order_demands_tenant ON sales_order_demands (enterprise_id, item_code);
CREATE INDEX IF NOT EXISTS idx_configured_rules_tenant ON configured_item_rules (enterprise_id, item_code);
CREATE INDEX IF NOT EXISTS idx_mrp_suggestions_tenant ON mrp_planned_suggestions (enterprise_id, plan_code);
CREATE INDEX IF NOT EXISTS idx_mrp_exceptions_tenant ON mrp_exception_messages (enterprise_id, plan_code);
CREATE INDEX IF NOT EXISTS idx_planning_params_tenant ON planning_params (enterprise_id, param_number);
CREATE INDEX IF NOT EXISTS idx_order_priorities_tenant ON order_priorities (enterprise_id, interval_start);
CREATE INDEX IF NOT EXISTS idx_item_machine_times_tenant ON item_machine_times (enterprise_id, item_code);
CREATE INDEX IF NOT EXISTS idx_kanban_cards_tenant ON kanban_cards (enterprise_id, item_code);
CREATE INDEX IF NOT EXISTS idx_mps_schedule_tenant ON mps_schedule (enterprise_id, item_code);
CREATE INDEX IF NOT EXISTS idx_item_planning_extras_tenant ON item_planning_extras (enterprise_id, item_code);
CREATE INDEX IF NOT EXISTS idx_machine_schedules_tenant ON machine_schedules (enterprise_id, machine_code, schedule_date);
CREATE INDEX IF NOT EXISTS idx_stock_balances_tenant ON stock_balances (enterprise_id, item_code);
CREATE INDEX IF NOT EXISTS idx_sales_divisions_tenant ON sales_divisions (enterprise_id, code);
