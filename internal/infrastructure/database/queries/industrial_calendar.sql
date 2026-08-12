-- name: CreateCalendarDay :one
INSERT INTO industrial_calendar (enterprise_id, year, month, day, is_workday, description, source)
VALUES (sqlc.arg(enterprise_id), sqlc.arg(year), sqlc.arg(month), sqlc.arg(day), sqlc.arg(is_workday), sqlc.arg(description), 'MANUAL')
    ON CONFLICT (enterprise_id, year, month, day) DO UPDATE SET is_workday = EXCLUDED.is_workday, description = EXCLUDED.description, source = 'MANUAL', updated_at = NOW()
                                          RETURNING *;

-- name: GetCalendarDay :one
SELECT * FROM industrial_calendar WHERE enterprise_id=sqlc.arg(enterprise_id) AND year=sqlc.arg(year) AND month=sqlc.arg(month) AND day=sqlc.arg(day);

-- name: GetWorkdaysInMonth :many
SELECT * FROM industrial_calendar WHERE enterprise_id=sqlc.arg(enterprise_id) AND year=sqlc.arg(year) AND month=sqlc.arg(month) AND is_workday = TRUE ORDER BY day;

-- name: IsWorkday :one
SELECT is_workday FROM industrial_calendar WHERE enterprise_id=sqlc.arg(enterprise_id) AND year=sqlc.arg(year) AND month=sqlc.arg(month) AND day=sqlc.arg(day);

-- name: GetNextWorkday :one
SELECT year, month, day FROM industrial_calendar
WHERE enterprise_id=sqlc.arg(enterprise_id) AND is_workday = TRUE AND (year > sqlc.arg(year) OR (year = sqlc.arg(year) AND month > sqlc.arg(month)) OR (year = sqlc.arg(year) AND month = sqlc.arg(month) AND day > sqlc.arg(day)))
ORDER BY year, month, day LIMIT 1;

-- name: ListCalendarMonth :many
SELECT * FROM industrial_calendar WHERE enterprise_id=sqlc.arg(enterprise_id) AND year=sqlc.arg(year) AND month=sqlc.arg(month) ORDER BY day;

-- name: DeleteCalendarDay :exec
DELETE FROM industrial_calendar WHERE enterprise_id=sqlc.arg(enterprise_id) AND year=sqlc.arg(year) AND month=sqlc.arg(month) AND day=sqlc.arg(day);

-- name: BatchInsertCalendarDays :copyfrom
INSERT INTO industrial_calendar (enterprise_id, year, month, day, is_workday, description, source)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GenerateCalendarMonth :execrows
INSERT INTO industrial_calendar (enterprise_id, year, month, day, is_workday, description, source)
SELECT sqlc.arg(enterprise_id), EXTRACT(YEAR FROM d)::int, EXTRACT(MONTH FROM d)::int, EXTRACT(DAY FROM d)::int,
       EXTRACT(ISODOW FROM d) BETWEEN 1 AND 5,
       CASE WHEN EXTRACT(ISODOW FROM d) BETWEEN 1 AND 5 THEN NULL ELSE 'Fim de semana' END,
       CASE WHEN EXTRACT(ISODOW FROM d) BETWEEN 1 AND 5 THEN 'AUTOMATICO' ELSE 'FIM_DE_SEMANA' END
FROM generate_series(
    make_date(sqlc.arg(year)::int, sqlc.arg(month)::int, 1),
    (make_date(sqlc.arg(year)::int, sqlc.arg(month)::int, 1) + interval '1 month - 1 day')::date,
    interval '1 day'
) d
ON CONFLICT (enterprise_id, year, month, day) DO NOTHING;
