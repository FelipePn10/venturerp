-- name: CreateModifier :one
INSERT INTO modifier (
    description,
    enterprise_id,
    created_by
) VALUES (
$1, $2, $3
)
RETURNING *;

-- name: GetModifierByID :one
SELECT * FROM modifier WHERE enterprise_id = $1 AND id = $2;

-- name: ListModifiers :many
SELECT * FROM modifier WHERE enterprise_id = $1 ORDER BY id;

-- name: UpdateModifier :one
UPDATE modifier
SET description = $3
WHERE enterprise_id = $1 AND id = $2
RETURNING *;
