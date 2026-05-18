-- name: InsertItemtMask :one
INSERT INTO item_masks (
    item_code,
    mask,
    mask_hash,
    created_by,
    created_at
)
VALUES ($1, $2, $3, $4, NOW())
RETURNING id, item_code, mask, mask_hash, created_by, created_at;

-- name: InsertItemMaskAnswer :exec
INSERT INTO item_mask_answers (mask_id, question_id, option_id, position)
VALUES ($1, $2, $3, $4);

-- name: GetOptionValueByID :one
SELECT value FROM question_options WHERE id = $1;

-- name: GetOptionIDByQuestionAndValue :one
SELECT id FROM question_options
WHERE question_id = $1 AND LOWER(value) = LOWER($2)
LIMIT 1;

-- name: GetItemQuestionPositions :many
SELECT question_id, position FROM item_questions
WHERE item_code = $1
ORDER BY position;
