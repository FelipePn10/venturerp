-- ─── Fiscal Classifications ───────────────────────────────────────────────────

-- name: CreateFiscalClassification :one
INSERT INTO fiscal_classifications (
    code, description, ncm, cest,
    ipi_rate, ipi_indicator, cst_ipi_entrada, cst_ipi_saida,
    pis_rate, pis_indicator, cst_pis_entrada, cst_pis_saida,
    cofins_rate, cofins_indicator, cst_cofins_entrada, cst_cofins_saida, cofins_majorado_pct,
    pis_st_pct, cofins_st_pct,
    pis_consumo_pct, cst_pis_consumo_entrada, cst_pis_consumo_saida,
    cofins_consumo_pct, cst_cofins_consumo_entrada, cst_cofins_consumo_saida,
    pis_retencao_pct, cst_pis_retencao, cofins_retencao_pct, cst_cofins_retencao,
    pis_reducao_pct, cst_pis_reducao, cofins_reducao_pct, cst_cofins_reducao,
    desc_pis_zf_pct, desc_cofins_zf_pct,
    ex_tarifario, un_ipi, un_tributacao, mod_bc_icms, mod_bc_icms_st,
    cod_clas_trib, cod_clas_trib_trib_reg, obs_fiscal, item_code, created_by,
    enterprise_id, valid_from, valid_until, default_origin, default_icms_rate, default_calculate_pis_cofins
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8,
    $9, $10, $11, $12,
    $13, $14, $15, $16, $17,
    $18, $19,
    $20, $21, $22,
    $23, $24, $25,
    $26, $27, $28, $29,
    $30, $31, $32, $33,
    $34, $35,
    $36, $37, $38, $39, $40,
    $41, $42, $43, $44, $45,
    sqlc.arg(enterprise_id), COALESCE(sqlc.narg(valid_from)::date, CURRENT_DATE),
    sqlc.narg(valid_until)::date, sqlc.narg(default_origin)::smallint,
    sqlc.narg(default_icms_rate)::numeric, sqlc.narg(default_calculate_pis_cofins)::boolean
) RETURNING *;

-- name: UpdateFiscalClassification :one
UPDATE fiscal_classifications SET
    description = $2, ncm = $3, cest = $4,
    ipi_rate = $5, ipi_indicator = $6, cst_ipi_entrada = $7, cst_ipi_saida = $8,
    pis_rate = $9, pis_indicator = $10, cst_pis_entrada = $11, cst_pis_saida = $12,
    cofins_rate = $13, cofins_indicator = $14, cst_cofins_entrada = $15, cst_cofins_saida = $16, cofins_majorado_pct = $17,
    pis_st_pct = $18, cofins_st_pct = $19,
    pis_consumo_pct = $20, cst_pis_consumo_entrada = $21, cst_pis_consumo_saida = $22,
    cofins_consumo_pct = $23, cst_cofins_consumo_entrada = $24, cst_cofins_consumo_saida = $25,
    pis_retencao_pct = $26, cst_pis_retencao = $27, cofins_retencao_pct = $28, cst_cofins_retencao = $29,
    pis_reducao_pct = $30, cst_pis_reducao = $31, cofins_reducao_pct = $32, cst_cofins_reducao = $33,
    desc_pis_zf_pct = $34, desc_cofins_zf_pct = $35,
    ex_tarifario = $36, un_ipi = $37, un_tributacao = $38, mod_bc_icms = $39, mod_bc_icms_st = $40,
    cod_clas_trib = $41, cod_clas_trib_trib_reg = $42, obs_fiscal = $43, item_code = $44, is_active = $45,
    valid_from = COALESCE(sqlc.narg(valid_from)::date, valid_from),
    valid_until = sqlc.narg(valid_until)::date, default_origin = sqlc.narg(default_origin)::smallint,
    default_icms_rate = sqlc.narg(default_icms_rate)::numeric,
    default_calculate_pis_cofins = sqlc.narg(default_calculate_pis_cofins)::boolean,
    updated_at = NOW()
WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id)
RETURNING *;

-- name: GetFiscalClassificationByCode :one
SELECT * FROM fiscal_classifications WHERE enterprise_id = $1 AND code = $2;

-- name: ListFiscalClassifications :many
SELECT * FROM fiscal_classifications
WHERE enterprise_id = $1 AND ($2::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextFiscalClassificationCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM fiscal_classifications WHERE enterprise_id = $1;

-- ─── Languages ────────────────────────────────────────────────────────────────

-- name: CreateFiscalClassificationLanguage :one
INSERT INTO fiscal_classification_languages (classification_id, language, description)
VALUES ($1, $2, $3)
ON CONFLICT (classification_id, language) DO UPDATE SET description = EXCLUDED.description
RETURNING *;

-- name: ListFiscalClassificationLanguages :many
SELECT * FROM fiscal_classification_languages WHERE classification_id = $1 ORDER BY language;

-- name: DeleteFiscalClassificationLanguage :exec
DELETE FROM fiscal_classification_languages WHERE id = $1;

-- ─── Export attributes ──────────────────────────────────────────────────────

-- name: CreateFiscalClassificationExportAttribute :one
INSERT INTO fiscal_classification_export_attributes (classification_id, code, description, domain, start_date, end_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListFiscalClassificationExportAttributes :many
SELECT * FROM fiscal_classification_export_attributes WHERE classification_id = $1 ORDER BY code;

-- name: DeleteFiscalClassificationExportAttribute :exec
DELETE FROM fiscal_classification_export_attributes WHERE id = $1;
