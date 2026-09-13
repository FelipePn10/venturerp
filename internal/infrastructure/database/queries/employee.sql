-- Toda consulta filtra por empresa (migração 343). Antes nenhuma filtrava, e
-- `UpdateEmployee`/`DeactivateEmployee` localizam por `code`: uma empresa
-- alterava ou inativava o funcionário da outra.

-- name: CreateNewEmployee :one
INSERT INTO employees (code, name, situation, participates_budget, technical_assistant, role, created_by, enterprise_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, sqlc.arg(enterprise_id))
RETURNING *;

-- name: UpdateEmployee :one
UPDATE employees
SET name               = $2,
    situation          = $3,
    participates_budget = $4,
    technical_assistant = $5,
    role               = $6,
    updated_at         = NOW()
WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id)
RETURNING *;

-- name: GetEmployeeByCode :one
SELECT * FROM employees WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: ListEmployees :many
SELECT * FROM employees WHERE enterprise_id = sqlc.arg(enterprise_id) ORDER BY code;

-- name: ListEmployeesByRole :many
SELECT * FROM employees WHERE role = $1 AND enterprise_id = sqlc.arg(enterprise_id) ORDER BY code;

-- name: DeactivateEmployee :exec
UPDATE employees SET situation = 'INACTIVE', updated_at = NOW()
WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id);
