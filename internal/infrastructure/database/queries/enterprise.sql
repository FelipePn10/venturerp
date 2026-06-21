-- name: CreateEnterprise :one
INSERT INTO enterprise (
    code,
    name,
    created_by,
    created_at
) VALUES (
    $1, $2, $3, now()
) RETURNING *;
