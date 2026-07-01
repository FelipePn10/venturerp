-- name: CreateGroup :one
INSERT INTO groups (
    code,
    description,
    enterprise_id,
    created_by
) VALUES (
$1, $2, $3, $4
)
RETURNING *;

-- name: GetGroupByCode :one
SELECT * FROM groups WHERE code = $1;

-- name: ListGroups :many
SELECT * FROM groups ORDER BY code;

-- name: UpdateGroup :one
UPDATE groups
SET description = $2, enterprise_id = $3
WHERE code = $1
RETURNING *;
