-- ─── State Groups ─────────────────────────────────────────────────────────────

-- name: CreateStateGroup :one
INSERT INTO state_groups (code, description, created_by)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetStateGroupByCode :one
SELECT * FROM state_groups WHERE code = $1;

-- name: ListStateGroups :many
SELECT * FROM state_groups ORDER BY code;

-- name: NextStateGroupCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM state_groups;

-- name: AddStateGroupUF :exec
INSERT INTO state_group_ufs (state_group_code, uf)
VALUES ($1, $2)
ON CONFLICT (state_group_code, uf) DO NOTHING;

-- name: ListStateGroupUFs :many
SELECT uf FROM state_group_ufs WHERE state_group_code = $1 ORDER BY uf;

-- name: UFInStateGroup :one
SELECT EXISTS (
    SELECT 1 FROM state_group_ufs WHERE state_group_code = $1 AND uf = $2
) AS in_group;

-- ─── Entry Operation Types ────────────────────────────────────────────────────

-- name: CreateEntryOperationType :one
INSERT INTO entry_operation_types (
    code, description, invoice_type_code, nature_operation,
    classification_type, classification_code, state_group_code, supplier_type_code, created_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateEntryOperationType :one
UPDATE entry_operation_types SET
    description = $2, invoice_type_code = $3, nature_operation = $4,
    classification_type = $5, classification_code = $6, state_group_code = $7,
    supplier_type_code = $8, is_active = $9
WHERE code = $1
RETURNING *;

-- name: GetEntryOperationTypeByCode :one
SELECT * FROM entry_operation_types WHERE code = $1;

-- name: ListEntryOperationTypes :many
SELECT * FROM entry_operation_types
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextEntryOperationTypeCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM entry_operation_types;
