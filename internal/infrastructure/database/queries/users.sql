-- name: CreateUser :exec
WITH created_user AS (
    INSERT INTO users (id, name, email, password, created_at, updated_at)
    VALUES ($1, $2, $3, $4, now(), now())
    RETURNING id
)
INSERT INTO user_enterprises (user_id, enterprise_id)
SELECT created_user.id, (SELECT id FROM enterprise WHERE code = @enterprise_code)
FROM created_user;

-- name: GetUserByEmail :one
SELECT
    id,
    name,
    email,
    password,
    role,
    auth_version,
    created_at,
    updated_at
FROM users
WHERE email = $1;

-- name: GetUserEnterpriseByCode :one
SELECT e.id
FROM user_enterprises ue
JOIN enterprise e ON e.id = ue.enterprise_id
WHERE ue.user_id = $1 AND e.code = $2;

-- name: GetOnlyUserEnterprise :one
SELECT MIN(enterprise_id)::bigint AS enterprise_id
FROM user_enterprises
WHERE user_id = $1
HAVING COUNT(*) = 1;

-- name: GetUserByID :one
SELECT
    id,
    name,
    email,
    role,
    created_at,
    updated_at
FROM users
WHERE id = $1;

-- name: GetUserAuthVersion :one
SELECT auth_version FROM users WHERE id = $1;
