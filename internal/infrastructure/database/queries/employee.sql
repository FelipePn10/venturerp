-- name: CreateNewEmployee :one
INSERT INTO employees (code, name, situation, participates_budget, technical_assistant, role, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateEmployee :one
UPDATE employees
SET name               = $2,
    situation          = $3,
    participates_budget = $4,
    technical_assistant = $5,
    role               = $6,
    updated_at         = NOW()
WHERE code = $1
RETURNING *;

-- name: GetEmployeeByCode :one
SELECT * FROM employees WHERE code = $1;

-- name: ListEmployees :many
SELECT * FROM employees ORDER BY code;

-- name: ListEmployeesByRole :many
SELECT * FROM employees WHERE role = $1 ORDER BY code;

-- name: DeactivateEmployee :exec
UPDATE employees SET situation = 'INACTIVE', updated_at = NOW() WHERE code = $1;
