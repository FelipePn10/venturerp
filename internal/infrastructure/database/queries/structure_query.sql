-- name: GetItemByCode :one
-- Retorna apenas os campos necessários para o resolver.
SELECT
    code,
    inherit
FROM items
WHERE code = $1;

-- name: GetMaskAnswersByItemAndValue :many
-- JOIN obrigatório: sem option_value a propagação gera "3#7" ao invés de "1.94M#1.94M".
-- Nome diferente de GetItemMaskAnswersByValue (sem option_value) em structure.sql.
SELECT
    ima.question_id,
    ima.option_id,
    ima.position,
    qo.value AS option_value
FROM item_masks im
         JOIN item_mask_answers ima ON ima.mask_id = im.id
         JOIN question_options   qo ON qo.id      = ima.option_id
WHERE im.item_code = $1
  AND im.mask      = $2
ORDER BY ima.position;