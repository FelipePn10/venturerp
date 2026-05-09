ALTER TABLE items
    RENAME COLUMN warehouse_id TO warehouse_code;

ALTER TABLE items
    RENAME COLUMN pdm_group_id TO pdm_group_code;

ALTER TABLE items
    RENAME COLUMN pdm_modifier_id TO pdm_modifier_code;

ALTER TABLE items
    RENAME COLUMN engineering_item_base_cod TO engineering_item_base_code;

ALTER TABLE items
    RENAME COLUMN planning_tank_id TO planning_tank_code;

ALTER TABLE items
    RENAME COLUMN planner_employee_id TO planner_employee_code;


ALTER TABLE items
ALTER COLUMN warehouse_code TYPE bigint
        USING warehouse_code::bigint;

ALTER TABLE items
ALTER COLUMN pdm_group_code TYPE bigint
        USING pdm_group_code::bigint;

ALTER TABLE items
ALTER COLUMN pdm_modifier_code TYPE bigint
        USING pdm_modifier_code::bigint;

ALTER TABLE items
ALTER COLUMN engineering_item_base_code TYPE bigint
        USING engineering_item_base_code::bigint;

ALTER TABLE items
ALTER COLUMN planning_tank_code TYPE bigint
        USING planning_tank_code::bigint;

ALTER TABLE items
ALTER COLUMN planner_employee_code TYPE bigint
        USING planner_employee_code::bigint;


CREATE INDEX IF NOT EXISTS idx_items_warehouse_code
    ON items (warehouse_code);

CREATE INDEX IF NOT EXISTS idx_items_pdm_group_code
    ON items (pdm_group_code);

CREATE INDEX IF NOT EXISTS idx_items_pdm_modifier_code
    ON items (pdm_modifier_code);

CREATE INDEX IF NOT EXISTS idx_items_engineering_item_base_code
    ON items (engineering_item_base_code);

CREATE INDEX IF NOT EXISTS idx_items_planning_tank_code
    ON items (planning_tank_code);

CREATE INDEX IF NOT EXISTS idx_items_planner_employee_code
    ON items (planner_employee_code);
