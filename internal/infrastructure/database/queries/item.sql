-- name: CreateItem :one
INSERT INTO items (
    enterprise_id,
    business_code,
    warehouse_code,
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
	commercial_description, commercial_sale_type, commercial_volume_conversion_factor,
	commercial_sale_multiple, commercial_minimum_sale_quantity, commercial_estimated_delivery_days,
	commercial_transfer_warehouse_code, commercial_technical_assistance_warehouse_code, commercial_packaging_item_code,
	commercial_allow_billing_description_change, commercial_issue_loading_labels, commercial_assemble_shipping_volumes,
	commercial_requires_special_packaging, commercial_withhold_pis_cofins, commercial_is_packaging,
	commercial_mobile_enabled, commercial_export_packaging, commercial_classification_code, commercial_notes,
	accounting_sale_fiscal_classification_code, accounting_purchase_fiscal_classification_code, accounting_origin,
	accounting_sale_ipi_type, accounting_sale_ipi_rate, accounting_purchase_ipi_type, accounting_purchase_ipi_rate,
	accounting_icms_rate, accounting_sale_unit_of_measurement, accounting_purchase_unit_of_measurement,
	accounting_inventory_group_code, accounting_classification_code, accounting_cest, accounting_input_code, accounting_notes,

    created_by,
    created_at
) VALUES (
             sqlc.arg(enterprise_id), sqlc.arg(business_code), sqlc.arg(warehouse_code), sqlc.arg(name), sqlc.arg(complement),
             sqlc.arg(nature), sqlc.arg(situation), sqlc.arg(health), sqlc.arg(pdm_group_code), sqlc.arg(pdm_modifier_code),
             sqlc.arg(pdm_attributes), sqlc.arg(pdm_description_technique), sqlc.arg(warehouse_unit_of_measurement),
             sqlc.arg(warehouse_automatic_low), sqlc.arg(warehouse_cyclical_count_config), sqlc.arg(warehouse_minimum_stock),
             sqlc.arg(warehouse_avg_monthly_consumption_manual), sqlc.arg(engineering_item_base_code), sqlc.arg(engineering_weight),
             sqlc.arg(engineering_dimensions), sqlc.arg(engineering_type), sqlc.arg(engineering_type_struct), sqlc.arg(engineering_oem),
             sqlc.arg(planning_type_mrp), sqlc.arg(planning_llc), sqlc.arg(planning_reorder_point), sqlc.arg(planning_tank_code),
             sqlc.arg(planning_ghost), sqlc.arg(planning_abc_class), sqlc.arg(planning_minimum_lot), sqlc.arg(planning_multiple_lot),
             sqlc.arg(planning_safety_stock), sqlc.arg(planning_critical), sqlc.arg(planning_exclusive), sqlc.arg(planning_active),
             sqlc.arg(supplies_type_of_use), sqlc.arg(supplies_purchase_uom), sqlc.arg(supplies_warehouse_code),
             sqlc.arg(supplies_receiving_checklist), sqlc.arg(supplies_harvest), sqlc.arg(commercial_warranty_days),
             TRUE, sqlc.narg(accounting_calculate_pis_cofins),
			 sqlc.arg(commercial_description), sqlc.arg(commercial_sale_type), sqlc.arg(commercial_volume_conversion_factor),
			 sqlc.arg(commercial_sale_multiple), sqlc.arg(commercial_minimum_sale_quantity), sqlc.arg(commercial_estimated_delivery_days),
			 sqlc.arg(commercial_transfer_warehouse_code), sqlc.arg(commercial_technical_assistance_warehouse_code), sqlc.arg(commercial_packaging_item_code),
			 sqlc.arg(commercial_allow_billing_description_change), sqlc.arg(commercial_issue_loading_labels), sqlc.arg(commercial_assemble_shipping_volumes),
			 sqlc.arg(commercial_requires_special_packaging), sqlc.arg(commercial_withhold_pis_cofins), sqlc.arg(commercial_is_packaging),
			 sqlc.arg(commercial_mobile_enabled), sqlc.arg(commercial_export_packaging), sqlc.arg(commercial_classification_code), sqlc.arg(commercial_notes),
			 sqlc.arg(accounting_sale_fiscal_classification_code), sqlc.arg(accounting_purchase_fiscal_classification_code), sqlc.arg(accounting_origin),
			 sqlc.arg(accounting_sale_ipi_type), sqlc.arg(accounting_sale_ipi_rate), sqlc.arg(accounting_purchase_ipi_type), sqlc.arg(accounting_purchase_ipi_rate),
			 sqlc.arg(accounting_icms_rate), sqlc.arg(accounting_sale_unit_of_measurement), sqlc.arg(accounting_purchase_unit_of_measurement),
			 sqlc.arg(accounting_inventory_group_code), sqlc.arg(accounting_classification_code), sqlc.arg(accounting_cest), sqlc.arg(accounting_input_code), sqlc.arg(accounting_notes),
			 sqlc.arg(created_by), NOW()
         )
    RETURNING *;

-- name: UpdateItemCommercialAccounting :one
UPDATE items SET
 commercial_description=$2, commercial_sale_type=$3, commercial_volume_conversion_factor=$4,
 commercial_sale_multiple=$5, commercial_minimum_sale_quantity=$6, commercial_estimated_delivery_days=$7,
 commercial_warranty_days=$8, commercial_transfer_warehouse_code=$9, commercial_technical_assistance_warehouse_code=$10,
 commercial_packaging_item_code=$11, commercial_allow_billing_description_change=$12,
 commercial_issue_loading_labels=$13, commercial_assemble_shipping_volumes=$14,
 commercial_requires_special_packaging=$15, commercial_withhold_pis_cofins=$16,
 commercial_is_packaging=$17, commercial_mobile_enabled=$18, commercial_export_packaging=$19,
 commercial_classification_code=$20, commercial_notes=$21,
 accounting_sale_fiscal_classification_code=$22, accounting_purchase_fiscal_classification_code=$23,
 accounting_origin=$24, accounting_sale_ipi_type=$25, accounting_sale_ipi_rate=$26,
 accounting_purchase_ipi_type=$27, accounting_purchase_ipi_rate=$28, accounting_icms_rate=$29,
 accounting_sale_unit_of_measurement=$30, accounting_purchase_unit_of_measurement=$31,
 accounting_inventory_group_code=$32, accounting_classification_code=$33, accounting_cest=$34,
 accounting_input_code=$35, accounting_notes=$36,
	accounting_calculate_pis_cofins=sqlc.narg(accounting_calculate_pis_cofins)::boolean,
	warehouse_cyclical_count_config=sqlc.narg(warehouse_cyclical_count_config)::jsonb
WHERE business_code=$1 AND enterprise_id=$37
RETURNING *;

-- name: NextAutomaticItemBusinessCode :one
SELECT next_item_business_code(sqlc.arg(enterprise_id));


-- name: FindItemByBusinessCode :one
SELECT *
FROM items
WHERE business_code = sqlc.arg(business_code)
  AND enterprise_id = sqlc.arg(enterprise_id);

-- name: FindItemByCode :one
SELECT *
FROM items
WHERE code = sqlc.arg(code)
  AND enterprise_id = sqlc.arg(enterprise_id);


-- name: GetItemByID :one
SELECT *
FROM items
WHERE id = sqlc.arg(id) AND enterprise_id = sqlc.arg(enterprise_id);

-- name: ListItems :many
SELECT *
FROM items
WHERE enterprise_id = sqlc.arg(enterprise_id)
ORDER BY code;

-- name: ItemFiscalClassificationExists :one
SELECT EXISTS (SELECT 1 FROM fiscal_classifications WHERE enterprise_id=sqlc.arg(enterprise_id) AND is_active
 AND valid_from<=CURRENT_DATE AND (valid_until IS NULL OR valid_until>=CURRENT_DATE)
 AND (code::text = sqlc.arg(classification_code)::text OR ncm = sqlc.arg(classification_code)::text));

-- name: GetEffectiveItemFiscalDefaults :one
SELECT f.id,f.code,f.ncm,f.cest,f.default_origin,f.un_tributacao,f.un_ipi,f.ipi_rate,
 f.default_icms_rate,f.pis_rate,f.cofins_rate,f.default_calculate_pis_cofins
FROM fiscal_classifications f
WHERE f.enterprise_id=sqlc.arg(enterprise_id) AND f.is_active
 AND f.valid_from<=CURRENT_DATE AND (f.valid_until IS NULL OR f.valid_until>=CURRENT_DATE)
 AND (f.code::text=sqlc.arg(classification_code)::text OR f.ncm=sqlc.arg(classification_code)::text)
ORDER BY CASE WHEN f.code::text=sqlc.arg(classification_code)::text THEN 0 ELSE 1 END,f.valid_from DESC,f.id DESC LIMIT 1;
-- name: ValidateItemPDMReferences :one
SELECT ($2::bigint=0 OR EXISTS(SELECT 1 FROM groups g WHERE g.enterprise_id=$1 AND g.code=$2))
   AND ($3::bigint=0 OR EXISTS(SELECT 1 FROM modifier m WHERE m.enterprise_id=$1 AND m.id=$3)) AS valid;
