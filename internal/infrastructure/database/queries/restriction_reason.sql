-- name: CreateRestrictionReason :one
INSERT INTO restriction_reasons (description, situation)
VALUES ($1, $2)
RETURNING *;

-- name: GetRestrictionReasonByCode :one
SELECT * FROM restriction_reasons WHERE code = $1;

-- name: ListRestrictionReasons :many
SELECT * FROM restriction_reasons ORDER BY code;

-- name: UpdateRestrictionReason :one
UPDATE restriction_reasons
SET description = $2,
    situation   = $3,
    updated_at  = NOW()
WHERE code = $1
RETURNING *;

-- name: DeleteRestrictionReason :exec
DELETE FROM restriction_reasons WHERE code = $1;
