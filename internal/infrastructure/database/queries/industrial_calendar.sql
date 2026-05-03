-- name: CreateCalendarDay :one
INSERT INTO industrial_calendar (year, month, day, is_workday, description)
VALUES ($1, $2, $3, $4, $5)
    ON CONFLICT (year, month, day) DO UPDATE SET is_workday = EXCLUDED.is_workday, description = EXCLUDED.description
                                          RETURNING *;

-- name: GetCalendarDay :one
SELECT * FROM industrial_calendar WHERE year = $1 AND month = $2 AND day = $3;

-- name: GetWorkdaysInMonth :many
SELECT * FROM industrial_calendar WHERE year = $1 AND month = $2 AND is_workday = TRUE ORDER BY day;

-- name: IsWorkday :one
SELECT is_workday FROM industrial_calendar WHERE year = $1 AND month = $2 AND day = $3;

-- name: GetNextWorkday :one
SELECT year, month, day FROM industrial_calendar
WHERE is_workday = TRUE AND (year > $1 OR (year = $1 AND month > $2) OR (year = $1 AND month = $2 AND day > $3))
ORDER BY year, month, day LIMIT 1;

-- name: ListCalendarMonth :many
SELECT * FROM industrial_calendar WHERE year = $1 AND month = $2 ORDER BY day;

-- name: DeleteCalendarDay :exec
DELETE FROM industrial_calendar WHERE year = $1 AND month = $2 AND day = $3;

-- name: BatchInsertCalendarDays :copyfrom
INSERT INTO industrial_calendar (year, month, day, is_workday, description)
VALUES ($1, $2, $3, $4, $5);
