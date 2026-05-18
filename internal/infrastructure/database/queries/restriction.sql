-- name: CreateRestriction :one
INSERT INTO restrictions (
    situation, customer_code, item_code, reason_code, classification_type,
    classification_origin, division_id, weight, created_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateRestriction :one
UPDATE restrictions
SET situation             = $2,
    customer_code         = $3,
    item_code             = $4,
    reason_code           = $5,
    classification_type   = $6,
    classification_origin = $7,
    division_id           = $8,
    weight                = $9,
    updated_at            = NOW()
WHERE code = $1
RETURNING *;

-- name: GetRestrictionByCode :one
SELECT * FROM restrictions WHERE code = $1;

-- name: GetRestrictionsByItemCode :many
SELECT * FROM restrictions
WHERE item_code = $1 AND situation = 'ACTIVE'
ORDER BY weight DESC;

-- name: GetRestrictionsByCustomerCode :many
SELECT * FROM restrictions
WHERE customer_code = $1 AND situation = 'ACTIVE'
ORDER BY weight DESC;

-- name: ListActiveRestrictions :many
SELECT * FROM restrictions WHERE situation = 'ACTIVE' ORDER BY weight DESC;

-- name: ListRestrictions :many
SELECT * FROM restrictions ORDER BY code;

-- name: DeactivateRestriction :exec
UPDATE restrictions SET situation = 'INACTIVE', updated_at = NOW() WHERE code = $1;

-- name: AddRestrictionDominant :one
INSERT INTO restriction_dominants (
    restriction_id, question_id, operator, condition_type, answer_value, sequence
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: AddRestrictionDeterminant :one
INSERT INTO restriction_determinants (
    restriction_id, question_id, operator, answer_value
)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: DeleteRestrictionDominant :exec
DELETE FROM restriction_dominants WHERE id = $1;

-- name: DeleteRestrictionDeterminant :exec
DELETE FROM restriction_determinants WHERE id = $1;

-- name: GetRestrictionDominants :many
SELECT * FROM restriction_dominants WHERE restriction_id = $1 ORDER BY sequence;

-- name: GetRestrictionDeterminants :many
SELECT * FROM restriction_determinants WHERE restriction_id = $1;

-- name: ListActiveRestrictionsByItems :many
SELECT code, item_code FROM restrictions
WHERE item_code = ANY(@item_codes::bigint[])
  AND situation = 'ACTIVE';
