-- ─── Legal Devices ────────────────────────────────────────────────────────────

-- name: CreateLegalDevice :one
INSERT INTO legal_devices (code, type, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateLegalDevice :one
UPDATE legal_devices
SET type = $2, description = $3, is_active = $4
WHERE id = $1
RETURNING *;

-- name: GetLegalDeviceByCode :one
SELECT * FROM legal_devices WHERE code = $1;

-- name: ListLegalDevices :many
SELECT * FROM legal_devices
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: ListLegalDevicesByType :many
SELECT * FROM legal_devices
WHERE type = $1
  AND ($2::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextLegalDeviceCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM legal_devices;

-- ─── CFOP / Naturezas de Operação ─────────────────────────────────────────────

-- name: CreateCFOP :one
INSERT INTO cfops (
    code, description, description_full, utilization,
    origem_clas_ipi, ind_operacao, tipo_utilizacao,
    codigo_anexo_sn, difal, doacao
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateCFOP :one
UPDATE cfops
SET description = $2, description_full = $3, utilization = $4,
    origem_clas_ipi = $5, ind_operacao = $6, tipo_utilizacao = $7,
    codigo_anexo_sn = $8, difal = $9, doacao = $10, is_active = $11
WHERE id = $1
RETURNING *;

-- name: GetCFOPByCode :one
SELECT * FROM cfops WHERE code = $1;

-- name: ListCFOPs :many
SELECT * FROM cfops
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: ListCFOPsByDirection :many
-- direction: 'IN' → codes 1xxx/2xxx/3xxx; 'OUT' → codes 5xxx/6xxx/7xxx
SELECT * FROM cfops
WHERE ($1::TEXT = '' OR
       CASE $1::TEXT
           WHEN 'IN'  THEN code BETWEEN 1000 AND 3999
           WHEN 'OUT' THEN code BETWEEN 5000 AND 7999
           ELSE TRUE
       END)
  AND ($2::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- ─── ICMS/IPI Tax Params ──────────────────────────────────────────────────────

-- name: CreateTaxParam :one
INSERT INTO icms_ipi_tax_params (
    ncm_code, item_code, item_config_mask, uf, operation_type,
    customer_code, customer_establishment_code,
    market_segment_id, invoice_type_exit_id, invoice_type_entry_id, tax_type_id,
    is_preferred, is_simples_optante,
    icms_pct_contrib, legal_device_icms_contrib_id,
    icms_pct_non_contrib, legal_device_icms_non_contrib_id,
    icms_red_pct_contrib, icms_red_target_contrib, legal_device_icms_red_contrib_id,
    icms_red_pct_non_contrib, icms_red_target_non_contrib, legal_device_icms_red_non_contrib_id,
    icms_deferral_pct, icms_deferral_target, legal_device_icms_deferral_id, cod_benef_rbc,
    icms_subst_pct_contrib, legal_device_icms_subst_contrib_id,
    icms_subst_pct_non_contrib, legal_device_icms_subst_non_contrib_id,
    icms_subst_pct_contrib_uc, icms_subst_red_pct, legal_device_icms_subst_red_id,
    icms_internal_pct, bc_icms_st_modality, icms_pct_for_st_contrib, icms_pct_for_st_non_contrib,
    cst_situation_b, csosn_icms, cst_icms_contrib, cst_icms_non_contrib,
    cod_beneficio_fiscal, cst_icms_contrib_dev, cst_icms_non_contrib_dev,
    ipi_red_pct_contrib, ipi_red_target_contrib, legal_device_ipi_contrib_id,
    ipi_red_pct_non_contrib, ipi_red_target_non_contrib, legal_device_ipi_non_contrib_id,
    cst_ipi_exit, cst_ipi_entry,
    icms_pct_origins_1238, calc_base_red_fci, icms_subst_pct_origins_1238,
    cst_icms_fci, uses_icms_zona_franca, dif_aliq_st_contrib_uc,
    cod_benef_contrib, cod_benef_non_contrib,
    icms_acres_pct_contrib, icms_acres_type_contrib, icms_acres_sum_contrib,
    icms_acres_pct_non_contrib, icms_acres_type_non_contrib, icms_acres_sum_non_contrib,
    icms_st_acres_pct_contrib, icms_st_acres_type_contrib, icms_st_acres_sum_contrib,
    icms_st_acres_pct_non_contrib, icms_st_acres_type_non_contrib, icms_st_acres_sum_non_contrib,
    fcp_st_partilha_pct,
    icms_difal_red_pct, icms_difal_type,
    difal_purchase_red_pct, difal_purchase_red_target
) VALUES (
    $1,  $2,  $3,  $4,  $5,
    $6,  $7,  $8,  $9,  $10, $11,
    $12, $13,
    $14, $15, $16, $17,
    $18, $19, $20, $21, $22, $23,
    $24, $25, $26, $27,
    $28, $29, $30, $31, $32, $33, $34,
    $35, $36, $37, $38,
    $39, $40, $41, $42, $43, $44, $45,
    $46, $47, $48, $49, $50, $51, $52, $53,
    $54, $55, $56, $57, $58, $59, $60, $61,
    $62, $63, $64, $65, $66, $67,
    $68, $69, $70, $71, $72, $73, $74,
    $75, $76, $77, $78
)
RETURNING *;

-- name: UpdateTaxParam :one
UPDATE icms_ipi_tax_params
SET ncm_code = $2, item_code = $3, item_config_mask = $4, uf = $5, operation_type = $6,
    customer_code = $7, customer_establishment_code = $8,
    market_segment_id = $9, invoice_type_exit_id = $10, invoice_type_entry_id = $11, tax_type_id = $12,
    is_preferred = $13, is_simples_optante = $14,
    icms_pct_contrib = $15, legal_device_icms_contrib_id = $16,
    icms_pct_non_contrib = $17, legal_device_icms_non_contrib_id = $18,
    icms_red_pct_contrib = $19, icms_red_target_contrib = $20, legal_device_icms_red_contrib_id = $21,
    icms_red_pct_non_contrib = $22, icms_red_target_non_contrib = $23, legal_device_icms_red_non_contrib_id = $24,
    icms_deferral_pct = $25, icms_deferral_target = $26, legal_device_icms_deferral_id = $27, cod_benef_rbc = $28,
    icms_subst_pct_contrib = $29, legal_device_icms_subst_contrib_id = $30,
    icms_subst_pct_non_contrib = $31, legal_device_icms_subst_non_contrib_id = $32,
    icms_subst_pct_contrib_uc = $33, icms_subst_red_pct = $34, legal_device_icms_subst_red_id = $35,
    icms_internal_pct = $36, bc_icms_st_modality = $37, icms_pct_for_st_contrib = $38, icms_pct_for_st_non_contrib = $39,
    cst_situation_b = $40, csosn_icms = $41, cst_icms_contrib = $42, cst_icms_non_contrib = $43,
    cod_beneficio_fiscal = $44, cst_icms_contrib_dev = $45, cst_icms_non_contrib_dev = $46,
    ipi_red_pct_contrib = $47, ipi_red_target_contrib = $48, legal_device_ipi_contrib_id = $49,
    ipi_red_pct_non_contrib = $50, ipi_red_target_non_contrib = $51, legal_device_ipi_non_contrib_id = $52,
    cst_ipi_exit = $53, cst_ipi_entry = $54,
    icms_pct_origins_1238 = $55, calc_base_red_fci = $56, icms_subst_pct_origins_1238 = $57,
    cst_icms_fci = $58, uses_icms_zona_franca = $59, dif_aliq_st_contrib_uc = $60,
    cod_benef_contrib = $61, cod_benef_non_contrib = $62,
    icms_acres_pct_contrib = $63, icms_acres_type_contrib = $64, icms_acres_sum_contrib = $65,
    icms_acres_pct_non_contrib = $66, icms_acres_type_non_contrib = $67, icms_acres_sum_non_contrib = $68,
    icms_st_acres_pct_contrib = $69, icms_st_acres_type_contrib = $70, icms_st_acres_sum_contrib = $71,
    icms_st_acres_pct_non_contrib = $72, icms_st_acres_type_non_contrib = $73, icms_st_acres_sum_non_contrib = $74,
    fcp_st_partilha_pct = $75,
    icms_difal_red_pct = $76, icms_difal_type = $77,
    difal_purchase_red_pct = $78, difal_purchase_red_target = $79,
    is_active = $80, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetTaxParamByID :one
SELECT * FROM icms_ipi_tax_params WHERE id = $1;

-- name: ListTaxParams :many
SELECT * FROM icms_ipi_tax_params
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY id;

-- name: ListTaxParamsByUF :many
SELECT * FROM icms_ipi_tax_params
WHERE uf = $1
  AND ($2::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY id;

-- name: ListTaxParamsByItem :many
SELECT * FROM icms_ipi_tax_params
WHERE item_code = $1
  AND ($2::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY uf, operation_type;

-- name: ListTaxParamsByNCM :many
SELECT * FROM icms_ipi_tax_params
WHERE ncm_code = $1
  AND ($2::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY uf, operation_type;

-- name: NextTaxParamID :one
SELECT COALESCE(MAX(id), 0) + 1 AS next_id FROM icms_ipi_tax_params;

-- ─── Item Classification Masks ────────────────────────────────────────────────

-- name: CreateClassificationMask :one
INSERT INTO item_classification_masks (code, mask, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateClassificationMask :one
UPDATE item_classification_masks
SET description = $2, is_active = $3
WHERE id = $1
RETURNING *;

-- name: GetClassificationMaskByCode :one
SELECT * FROM item_classification_masks WHERE code = $1;

-- name: ListClassificationMasks :many
SELECT * FROM item_classification_masks
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextClassificationMaskCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM item_classification_masks;

-- ─── Item Classifications ─────────────────────────────────────────────────────

-- name: CreateItemClassification :one
INSERT INTO item_classifications (code, mask_id, parent_id, level, description)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateItemClassification :one
UPDATE item_classifications
SET description = $2, is_active = $3
WHERE id = $1
RETURNING *;

-- name: GetItemClassificationByCode :one
SELECT c.* FROM item_classifications c
JOIN item_classification_masks m ON m.id = c.mask_id
WHERE c.code = $1 AND m.code = $2;

-- name: ListItemClassificationsByMask :many
SELECT * FROM item_classifications
WHERE mask_id = $1
  AND ($2::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY level, code;

-- name: ListItemClassificationChildren :many
SELECT * FROM item_classifications
WHERE parent_id = $1
  AND ($2::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- ─── Countries ────────────────────────────────────────────────────────────────

-- name: CreateCountry :one
INSERT INTO countries (sigla, name, ddi, bacen_code, sis_comex)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateCountry :one
UPDATE countries
SET name = $2, ddi = $3, bacen_code = $4, sis_comex = $5, is_active = $6
WHERE id = $1
RETURNING *;

-- name: GetCountryBySigla :one
SELECT * FROM countries WHERE sigla = $1;

-- name: ListCountries :many
SELECT * FROM countries
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY sigla;

-- ─── UFs ──────────────────────────────────────────────────────────────────────

-- name: CreateUF :one
INSERT INTO ufs (sigla, name, country_id, ibge_code)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateUF :one
UPDATE ufs
SET name = $2, ibge_code = $3, is_active = $4
WHERE id = $1
RETURNING *;

-- name: GetUFBySigla :one
SELECT * FROM ufs WHERE sigla = $1;

-- name: ListUFs :many
SELECT * FROM ufs
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY sigla;

-- name: ListUFsByCountry :many
SELECT u.* FROM ufs u
JOIN countries c ON c.id = u.country_id
WHERE c.sigla = $1
  AND ($2::BOOLEAN = FALSE OR u.is_active = TRUE)
ORDER BY u.sigla;
