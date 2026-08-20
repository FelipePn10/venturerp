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
SELECT * FROM groups WHERE enterprise_id = $1 AND code = $2;

-- name: ListGroups :many
SELECT * FROM groups WHERE enterprise_id = $1 ORDER BY code;

-- name: UpdateGroup :one
UPDATE groups
SET description = $3
WHERE enterprise_id = $1 AND code = $2
RETURNING *;
