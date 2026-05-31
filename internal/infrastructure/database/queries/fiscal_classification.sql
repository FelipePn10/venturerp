-- ─── Fiscal Classifications ───────────────────────────────────────────────────

-- name: CreateFiscalClassification :one
INSERT INTO fiscal_classifications (
    code, description, ncm, cest,
    ipi_rate, ipi_indicator, apuracao, cst_ipi_entrada, cst_ipi_saida,
    pis_rate, pis_indicator, cst_pis_entrada, cst_pis_saida,
    cofins_rate, cofins_indicator, cst_cofins_entrada, cst_cofins_saida, cofins_majorado_pct,
    pis_st_pct, cofins_st_pct,
    pis_consumo_pct, cst_pis_consumo_entrada, cst_pis_consumo_saida,
    cofins_consumo_pct, cst_cofins_consumo_entrada, cst_cofins_consumo_saida,
    pis_retencao_pct, cst_pis_retencao, cofins_retencao_pct, cst_cofins_retencao,
    pis_reducao_pct, cst_pis_reducao, cofins_reducao_pct, cst_cofins_reducao,
    desc_pis_zf_pct, desc_cofins_zf_pct,
    ex_tarifario, un_ipi, un_tributacao, mod_bc_icms, mod_bc_icms_st,
    cod_clas_trib, cod_clas_trib_trib_reg, obs_fiscal, created_by
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8, $9,
    $10, $11, $12, $13,
    $14, $15, $16, $17, $18,
    $19, $20,
    $21, $22, $23,
    $24, $25, $26,
    $27, $28, $29, $30,
    $31, $32, $33, $34,
    $35, $36,
    $37, $38, $39, $40, $41,
    $42, $43, $44, $45
) RETURNING *;

-- name: UpdateFiscalClassification :one
UPDATE fiscal_classifications SET
    description = $2, ncm = $3, cest = $4,
    ipi_rate = $5, ipi_indicator = $6, apuracao = $7, cst_ipi_entrada = $8, cst_ipi_saida = $9,
    pis_rate = $10, pis_indicator = $11, cst_pis_entrada = $12, cst_pis_saida = $13,
    cofins_rate = $14, cofins_indicator = $15, cst_cofins_entrada = $16, cst_cofins_saida = $17, cofins_majorado_pct = $18,
    pis_st_pct = $19, cofins_st_pct = $20,
    pis_consumo_pct = $21, cst_pis_consumo_entrada = $22, cst_pis_consumo_saida = $23,
    cofins_consumo_pct = $24, cst_cofins_consumo_entrada = $25, cst_cofins_consumo_saida = $26,
    pis_retencao_pct = $27, cst_pis_retencao = $28, cofins_retencao_pct = $29, cst_cofins_retencao = $30,
    pis_reducao_pct = $31, cst_pis_reducao = $32, cofins_reducao_pct = $33, cst_cofins_reducao = $34,
    desc_pis_zf_pct = $35, desc_cofins_zf_pct = $36,
    ex_tarifario = $37, un_ipi = $38, un_tributacao = $39, mod_bc_icms = $40, mod_bc_icms_st = $41,
    cod_clas_trib = $42, cod_clas_trib_trib_reg = $43, obs_fiscal = $44, is_active = $45,
    updated_at = NOW()
WHERE code = $1
RETURNING *;

-- name: GetFiscalClassificationByCode :one
SELECT * FROM fiscal_classifications WHERE code = $1;

-- name: ListFiscalClassifications :many
SELECT * FROM fiscal_classifications
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextFiscalClassificationCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM fiscal_classifications;

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
