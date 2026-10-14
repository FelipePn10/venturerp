-- Grupos de UF e TES são catálogos da empresa (enterprise_id). O código se
-- repete entre empresas; a busca é sempre (empresa, código).

-- ─── State Groups ─────────────────────────────────────────────────────────────

-- name: CreateStateGroup :one
INSERT INTO state_groups (enterprise_id, code, description, created_by)
VALUES ($4, $1, $2, $3)
RETURNING *;

-- name: GetStateGroupByCode :one
SELECT * FROM state_groups WHERE code = $1 AND enterprise_id = $2;

-- name: ListStateGroups :many
SELECT * FROM state_groups WHERE enterprise_id = $1 ORDER BY code;

-- name: NextStateGroupCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM state_groups WHERE enterprise_id = $1;

-- name: AddStateGroupUF :execrows
INSERT INTO state_group_ufs (enterprise_id, state_group_code, uf)
SELECT $3, $1, $2
 WHERE EXISTS (SELECT 1 FROM state_groups g WHERE g.code = $1 AND g.enterprise_id = $3)
ON CONFLICT (enterprise_id, state_group_code, uf) DO NOTHING;

-- name: ListStateGroupUFs :many
SELECT uf FROM state_group_ufs WHERE state_group_code = $1 AND enterprise_id = $2 ORDER BY uf;

-- name: UFInStateGroup :one
SELECT EXISTS (
    SELECT 1 FROM state_group_ufs WHERE state_group_code = $1 AND uf = $2 AND enterprise_id = $3
) AS in_group;

-- ─── Entry Operation Types ────────────────────────────────────────────────────

-- name: CreateEntryOperationType :one
INSERT INTO entry_operation_types (
    enterprise_id, code, description, invoice_type_code, nature_operation,
    classification_type, classification_code, state_group_code, supplier_type_code, created_by,
    movimenta_estoque, gera_financeiro, credita_icms, credita_ipi, credita_pis_cofins
)
VALUES ($15, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: UpdateEntryOperationType :one
UPDATE entry_operation_types SET
    description = $2, invoice_type_code = $3, nature_operation = $4,
    classification_type = $5, classification_code = $6, state_group_code = $7,
    supplier_type_code = $8, is_active = $9,
    movimenta_estoque = $10, gera_financeiro = $11, credita_icms = $12,
    credita_ipi = $13, credita_pis_cofins = $14
WHERE code = $1 AND enterprise_id = $15
RETURNING *;

-- name: GetEntryOperationTypeByCode :one
SELECT * FROM entry_operation_types WHERE code = $1 AND enterprise_id = $2;

-- name: ListEntryOperationTypes :many
SELECT * FROM entry_operation_types
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE) AND enterprise_id = $2
ORDER BY code;

-- name: NextEntryOperationTypeCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM entry_operation_types WHERE enterprise_id = $1;
