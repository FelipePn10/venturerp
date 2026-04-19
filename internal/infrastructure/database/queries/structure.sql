-- name: CreateStructureComponent :one
INSERT INTO item_structures (
    parent_code,
    child_code,
    parent_mask,
    quantity,
    unit_of_measurement,
    loss_percentage,
    position,
    health,
    notes,
    created_by
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
         )
    RETURNING *;


-- name: GetStructureComponentByID :one
SELECT *
FROM item_structures
WHERE id = $1;


-- name: GetAllDirectChildren :many
SELECT
    s.id,
    s.parent_code,
    s.child_code,
    i.pdm_description_technique AS child_description,
    s.parent_mask,
    s.quantity,
    s.loss_percentage,
    s.unit_of_measurement,
    s.health,
    s.position,
    s.notes,
    s.is_active,
    s.created_by,
    s.created_at,
    s.updated_at
FROM item_structures s
         JOIN items i ON i.code = s.child_code
WHERE s.parent_code = $1
  AND s.is_active = TRUE
ORDER BY s.position, s.id;


-- name: GetGenericChildren :many
SELECT *
FROM item_structures
WHERE parent_code = $1
  AND parent_mask IS NULL
  AND is_active = TRUE
ORDER BY position, id;


-- name: GetDirectChildrenForMask :many
SELECT *
FROM item_structures
WHERE parent_code = $1
  AND is_active = TRUE
  AND (parent_mask = $2 OR parent_mask IS NULL)
ORDER BY
    CASE WHEN parent_mask IS NOT NULL THEN 0 ELSE 1 END,
    position,
    id;


-- name: UpdateStructureComponent :one
UPDATE item_structures
SET
    quantity            = $2,
    unit_of_measurement = $3,
    loss_percentage     = $4,
    position            = $5,
    health              = $6,
    notes               = $7,
    updated_at          = NOW()
WHERE id = $1
  AND is_active = TRUE
    RETURNING *;


-- name: DeactivateStructureComponent :exec
UPDATE item_structures
SET
    is_active  = FALSE,
    updated_at = NOW()
WHERE id = $1;


-- name: GetItemCodeAndDescription :one
SELECT
    i.code::BIGINT AS code,
    i.pdm_description_technique AS description
FROM items i
WHERE i.code = $1
    LIMIT 1;

-- name: ItemExists :one
SELECT EXISTS (
    SELECT 1
    FROM items
    WHERE code = $1
) AS "exists";

-- name: HasCyclicReference :one
SELECT has_cycle($1, $2) AS has_cycle;

-- name: GetItemMaskAnswersByValue :many
SELECT
    ima.question_id,
    ima.position,
    ima.option_id
FROM item_masks im
         JOIN item_mask_answers ima ON ima.mask_id = im.id
WHERE im.item_code = $1
  AND im.mask = $2
ORDER BY ima.position;


-- name: GetItemQuestions :many
SELECT
    iq.question_id,
    iq.position
FROM item_questions iq
WHERE iq.item_id = $1
ORDER BY iq.position;