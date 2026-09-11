-- name: CreateRestrictionReason :one
INSERT INTO restriction_reasons (description, situation, enterprise_id)
VALUES ($1, $2, sqlc.arg(enterprise_id))
RETURNING *;

-- name: GetRestrictionReasonByCode :one
SELECT * FROM restriction_reasons
WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: ListRestrictionReasons :many
SELECT * FROM restriction_reasons
WHERE enterprise_id = sqlc.arg(enterprise_id)
ORDER BY code;

-- name: UpdateRestrictionReason :one
UPDATE restriction_reasons
SET description = $2,
    situation   = $3,
    updated_at  = NOW()
WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id)
RETURNING *;

-- name: DeleteRestrictionReason :execrows
DELETE FROM restriction_reasons
WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id);
