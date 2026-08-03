-- name: CreateItem :one
INSERT INTO items (
    warehouse_code,
    code,
    name,
    complement,
    nature,
    situation,
    health,

    pdm_group_code,
    pdm_modifier_code,
    pdm_attributes,
    pdm_description_technique,

    warehouse_unit_of_measurement,
    warehouse_automatic_low,
    warehouse_cyclical_count_config,
    warehouse_minimum_stock,
    warehouse_avg_monthly_consumption_manual,

    engineering_item_base_code,
    engineering_weight,
    engineering_dimensions,
    engineering_type,
    engineering_type_struct,
    engineering_oem,

    planning_type_mrp,
    planning_llc,
    planning_reorder_point,
    planning_tank_code,
    planning_ghost,
	planning_abc_class,
	planning_minimum_lot,
	planning_multiple_lot,
	planning_safety_stock,
	planning_critical,
	planning_exclusive,
	planning_active,

    supplies_type_of_use,
	supplies_purchase_uom,
	supplies_warehouse_code,
	supplies_receiving_checklist,
	supplies_harvest,

	commercial_warranty_days,
	accounting_active,
	accounting_calculate_pis_cofins,

    created_by,
    created_at
) VALUES (
             $1,  $2,  $3,  $4,
             $5,  $6,  $7,  $8,  $9,
             $10, $11, $12, $13, $14,
             $15, $16, $17, $18, $19,
             $20, $21, $22, $23, $24,
             $25, $26, $27, $28, $29,
			 $30, $31, $32, $33, $34,
			 $35, $36, $37, $38, $39,
			 $40, $41, $42, $43, NOW()
         )
    RETURNING *;


-- name: FindItemByCode :one
SELECT *
FROM items
WHERE code = $1;


-- name: GetItemByID :one
SELECT *
FROM items
WHERE id = $1;

-- name: ListItems :many
SELECT *
FROM items
ORDER BY code;
