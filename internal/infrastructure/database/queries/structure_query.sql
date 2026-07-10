-- name: GetWhereUsed :many
-- Implosão de estrutura: retorna todos os pais que utilizam o item $1,
-- recursivamente até o nível $2 (0 = todos).
WITH RECURSIVE where_used AS (
    SELECT
        s.parent_code,
        s.child_code,
        s.quantity,
        s.loss_percentage,
        s.parent_mask,
        s.sequence,
        s.is_active,
        1 AS level
    FROM item_structures s
    WHERE s.child_code = $1
      AND s.is_active = TRUE

    UNION ALL

    SELECT
        s.parent_code,
        s.child_code,
        s.quantity,
        s.loss_percentage,
        s.parent_mask,
        s.sequence,
        s.is_active,
        wu.level + 1
    FROM item_structures s
    JOIN where_used wu ON wu.parent_code = s.child_code
    WHERE s.is_active = TRUE
      AND ($2::int = 0 OR wu.level < $2::int)
)
SELECT
    wu.level,
    wu.parent_code,
    wu.child_code,
    p.pdm_description_technique AS parent_description,
    wu.quantity,
    wu.loss_percentage,
    wu.parent_mask,
    wu.sequence
FROM where_used wu
JOIN items p ON p.code = wu.parent_code
ORDER BY wu.level, wu.parent_code, wu.sequence;

-- name: GetChildrenForConsult :many
-- Consulta de estrutura VENG0401: retorna filhos diretos com campos de data,
-- fórmula de perda e dados do item filho (almoxarifado, tipo de estrutura).
-- $1 = parent_code, $2 = parent_mask (NULL → apenas genéricos),
-- $3 = effectiveness_date (NULL → sem filtro de data).
SELECT
    s.id,
    s.parent_code,
    s.child_code,
    i.pdm_description_technique AS child_description,
    s.parent_mask,
    s.quantity,
    s.loss_percentage,
    s.loss_formula,
    s.unit_of_measurement,
    s.health,
    s.sequence,
    s.notes,
    s.is_active,
    s.created_by,
    s.created_at,
    s.updated_at,
    s.inherit,
    s.start_date,
    s.end_date,
    i.warehouse_code,
    i.engineering_type_struct
FROM item_structures s
         JOIN items i ON i.code = s.child_code
WHERE s.parent_code = $1
  AND s.is_active = TRUE
  AND (s.parent_mask IS NULL OR s.parent_mask = $2)
  AND (s.start_date IS NULL OR $3::date IS NULL OR s.start_date <= $3::date)
  AND (s.end_date   IS NULL OR $3::date IS NULL OR s.end_date   >= $3::date)
ORDER BY
    CASE WHEN s.parent_mask IS NOT NULL THEN 0 ELSE 1 END,
    s.sequence,
    s.id;
