-- Registros do item são da empresa do item (items.enterprise_id).

-- name: UpsertItemCalendarDay :one
INSERT INTO item_calendar_promises (item_code, mask, year, month, day, is_workday, description)
SELECT $1, $2, $3, $4, $5, $6, $7
 WHERE EXISTS (SELECT 1 FROM items i WHERE i.code = $1 AND i.enterprise_id = $8)
    ON CONFLICT (item_code, mask, year, month, day) DO UPDATE SET is_workday = EXCLUDED.is_workday, description = EXCLUDED.description
                                                           RETURNING *;

-- name: GetItemCalendarDay :one
SELECT * FROM item_calendar_promises WHERE item_code = $1 AND mask = $2 AND year = $3 AND month = $4 AND day = $5
  AND EXISTS (SELECT 1 FROM items i WHERE i.code = item_calendar_promises.item_code AND i.enterprise_id = $6);

-- name: GetItemWorkdaysInMonth :many
SELECT * FROM item_calendar_promises WHERE item_code = $1 AND mask = $2 AND year = $3 AND month = $4 AND is_workday = TRUE
  AND EXISTS (SELECT 1 FROM items i WHERE i.code = item_calendar_promises.item_code AND i.enterprise_id = $5)
ORDER BY day;

-- name: ListItemCalendarMonth :many
SELECT * FROM item_calendar_promises WHERE item_code = $1 AND mask = $2 AND year = $3 AND month = $4
  AND EXISTS (SELECT 1 FROM items i WHERE i.code = item_calendar_promises.item_code AND i.enterprise_id = $5)
ORDER BY day;

-- name: DeleteItemCalendarDay :exec
DELETE FROM item_calendar_promises WHERE item_code = $1 AND mask = $2 AND year = $3 AND month = $4 AND day = $5
  AND EXISTS (SELECT 1 FROM items i WHERE i.code = item_calendar_promises.item_code AND i.enterprise_id = $6);
