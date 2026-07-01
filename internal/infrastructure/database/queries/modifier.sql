-- name: CreateModifier :one
INSERT INTO modifier (
    description,
    created_by
) VALUES (
$1, $2
)
RETURNING *;

-- name: GetModifierByID :one
SELECT * FROM modifier WHERE id = $1;

-- name: ListModifiers :many
SELECT * FROM modifier ORDER BY id;

-- name: UpdateModifier :one
UPDATE modifier
SET description = $2
WHERE id = $1
RETURNING *;
