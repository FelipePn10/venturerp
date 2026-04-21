-- name: AssociateQuestionItem :exec
INSERT INTO item_questions (
    item_code,
    question_id,
    position,
    created_at
) VALUES ($1, $2, $3, $4);

-- name: ExistsByItemAndQuestion :one
SELECT EXISTS (
    SELECT 1
    FROM item_questions
    WHERE item_code = $1
      AND question_id = $2
);

-- name: ExistsByItemAndPosition :one
SELECT EXISTS (
    SELECT 1
    FROM item_questions
    WHERE item_code = $1
      AND position = $2
);
