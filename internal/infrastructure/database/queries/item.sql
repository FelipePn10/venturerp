-- name: CreateItem :one
INSERT INTO items (
    warehouse_id,
    code,
    complement,
    nature,
    inherit,
    situation,
    health,
    pdm_group_id,
    pdm_modifier_id,
    pdm_attributes,
    pdm_description_technique,
    warehouse_unit_of_measurement,
    warehouse_automatic_low,
    warehouse_cyclical_count_config,
    warehouse_minimum_stock,
    warehouse_avg_monthly_consumption_manual,
    engineering_item_base_cod,
    engineering_weight,
    engineering_dimensions,
    engineering_type,
    engineering_type_struct,
    engineering_oem,
    planning_type_mrp,
    planning_llc,
    planning_reorder_point,
    planning_tank_id,
    planning_ghost,
    planner_employee_id,
    supplies_type_of_use,
    created_by,
    created_at
) VALUES (
    $1,  $2,  $3,  $4,  $5,
    $6,  $7,  $8,  $9,  $10,
    $11, $12, $13, $14, $15,
    $16, $17, $18, $19, $20,
    $21, $22, $23, $24, $25,
    $26, $27, $28, $29, $30, NOW()
)
RETURNING *;

-- name: CreateItemMachineUsage :one
INSERT INTO item_machine_usages (
    item_id,
    machine_id,
    usage_time
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: FindItemByCode :one
SELECT *
FROM items
WHERE code = $1;

-- name: GetItemByID :one
SELECT * FROM items
WHERE id = $1;

-- name: ListMachineUsagesByItem :many
SELECT * FROM item_machine_usages
WHERE item_id = $1;
