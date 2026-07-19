-- name: CreateUser :exec
WITH created_user AS (
    INSERT INTO users (id, name, email, password, created_at, updated_at)
    VALUES ($1, $2, $3, $4, now(), now())
    RETURNING id
)
INSERT INTO user_enterprises (user_id, enterprise_id)
SELECT created_user.id, @enterprise_id
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

-- name: GetUserAuthorizationByEnterpriseCode :one
SELECT e.id AS enterprise_id, e.code::bigint AS enterprise_code, ue.role, u.auth_version
FROM user_enterprises ue
JOIN enterprise e ON e.id = ue.enterprise_id
JOIN users u ON u.id = ue.user_id
WHERE ue.user_id = $1 AND e.code = $2;

-- name: GetOnlyUserAuthorization :one
SELECT MIN(ue.enterprise_id)::bigint AS enterprise_id,
	   MIN(e.code)::bigint AS enterprise_code,
       MIN(ue.role)::text AS role,
       MIN(u.auth_version)::bigint AS auth_version
FROM user_enterprises ue
JOIN users u ON u.id = ue.user_id
JOIN enterprise e ON e.id = ue.enterprise_id
WHERE ue.user_id = $1
HAVING COUNT(*) = 1;

-- name: GetCurrentUserAuthorization :one
SELECT ue.enterprise_id, e.code::bigint AS enterprise_code, ue.role, u.auth_version
FROM user_enterprises ue
JOIN users u ON u.id = ue.user_id
JOIN enterprise e ON e.id = ue.enterprise_id
WHERE ue.user_id = $1 AND ue.enterprise_id = $2;

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
