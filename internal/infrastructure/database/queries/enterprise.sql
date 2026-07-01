-- name: CreateEnterprise :one
INSERT INTO enterprise (
    code,
    name,
    created_by,
    created_at
) VALUES (
    $1, $2, $3, now()
) RETURNING *;

-- name: GetEnterpriseByID :one
SELECT * FROM enterprise WHERE id = $1;

-- name: GetEnterpriseByCode :one
SELECT * FROM enterprise WHERE code = $1;

-- name: ListEnterprises :many
SELECT * FROM enterprise ORDER BY code;
