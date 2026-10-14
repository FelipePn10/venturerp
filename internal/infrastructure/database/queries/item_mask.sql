-- Registros do item são da empresa do item (items.enterprise_id).

-- name: GetProductMaskByItemCode :one
SELECT *
FROM item_masks
WHERE item_code = $1
  AND EXISTS (SELECT 1 FROM items i WHERE i.code = item_masks.item_code AND i.enterprise_id = $2)
ORDER BY created_at DESC
LIMIT 1;

-- name: DeleteItemMask :exec
DELETE FROM item_masks
WHERE item_masks.id = $1
  AND EXISTS (SELECT 1 FROM items i WHERE i.code = item_masks.item_code AND i.enterprise_id = $2);

-- name: ListAllItemMasks :many
SELECT id, item_code, mask, mask_hash, created_by, created_at
FROM item_masks
WHERE EXISTS (SELECT 1 FROM items i WHERE i.code = item_masks.item_code AND i.enterprise_id = $1)
ORDER BY item_code, created_at DESC;
