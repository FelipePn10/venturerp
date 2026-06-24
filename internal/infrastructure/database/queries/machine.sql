-- name: CreateMachineType :one
INSERT INTO machine_types (
    code,
    name,
    description,
    type,
    requires_operator,
    is_active,
    created_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING *;

-- name: UpdateMachineType :one
UPDATE machine_types
SET
    name = $1,
    description = $2,
    type = $3,
    requires_operator = $4,
    is_active = $5,
    updated_at = NOW()
WHERE code = $6
    RETURNING *;

-- name: GetMachineTypeByCode :one
SELECT *
FROM machine_types
WHERE code = $1;

-- name: ListMachineTypes :many
SELECT *
FROM machine_types
WHERE is_active = TRUE
ORDER BY code;

-- name: DeleteMachineType :exec
UPDATE machine_types
SET is_active = FALSE, updated_at = NOW()
WHERE code = $1;

-- name: CreateMachine :one
INSERT INTO machines (
    code,
    name,
    machine_type_code,
    cost_center_code,
    capacity,
    capacity_unit,
    capacity_period,
    efficiency_rate,
    created_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    RETURNING *;

-- name: UpdateMachine :one
UPDATE machines
SET
    name = $1,
    machine_type_code = $2,
    cost_center_code = $3,
    capacity = $4,
    capacity_unit = $5,
    capacity_period = $6,
    efficiency_rate = $7,
    updated_at = NOW()
WHERE code = $6
    RETURNING *;

-- name: GetMachineByCode :one
SELECT *
FROM machines
WHERE code = $1;

-- name: ListMachines :many
SELECT *
FROM machines
WHERE is_active = TRUE
ORDER BY code;

-- name: ListMachinesByType :many
SELECT *
FROM machines
WHERE machine_type_code = $1
  AND is_active = TRUE
ORDER BY code;

-- name: DeleteMachine :exec
UPDATE machines
SET is_active = FALSE, updated_at = NOW()
WHERE code = $1;

-- name: CreateItemMachineTime :one
INSERT INTO item_machine_times (
    item_code,
    mask,
    machine_code,
    production_time,
    production_time_unit,
    production_base_qty,
    setup_time,
    priority
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    ON CONFLICT (item_code, mask, machine_code)
DO UPDATE SET
    production_time = EXCLUDED.production_time,
           setup_time = EXCLUDED.setup_time,
           priority = EXCLUDED.priority,
           updated_at = NOW()
           RETURNING *;


-- name: ListItemMachineTimes :many
SELECT *
FROM item_machine_times
WHERE item_code = $1
  AND is_active = TRUE
ORDER BY priority;

-- name: ListItemsByMachine :many
SELECT *
FROM item_machine_times
WHERE machine_code = $1
  AND is_active = TRUE
ORDER BY priority;

-- -- name: DeleteItemMachineTime :exec
-- UPDATE item_machine_times
-- SET is_active = FALSE, updated_at = NOW()
-- WHERE code = $1;

-- name: CreateSchedule :one
-- code is auto-assigned (MAX+1) so callers don't have to manage the business key;
-- the column is NOT NULL UNIQUE with no DB default.
INSERT INTO machine_schedules (
    code,
    machine_code,
    order_code,
    schedule_date,
    start_time,
    end_time,
    planned_qty,
    sequence,
    priority_override,
    notes
)
VALUES (
    COALESCE((SELECT MAX(code) FROM machine_schedules), 0) + 1,
    $1, $2, $3, $4, $5, $6, $7, $8, $9)
    RETURNING *;

-- name: GetSchedule :one
SELECT *
FROM machine_schedules
WHERE code = $1;

-- name: ListSchedules :many
SELECT *
FROM machine_schedules
WHERE machine_code = $1
  AND schedule_date = $2
  AND is_active = TRUE
ORDER BY sequence;

-- name: ListSchedulesByRange :many
SELECT *
FROM machine_schedules
WHERE machine_code = $1
  AND schedule_date BETWEEN $2 AND $3
  AND is_active = TRUE
ORDER BY schedule_date, sequence;

-- name: UpdateScheduleSequence :one
UPDATE machine_schedules
SET
    sequence = $2,
    priority_override = $3,
    updated_at = NOW()
WHERE code = $1
    RETURNING *;

-- name: UpdateScheduleStatus :one
UPDATE machine_schedules
SET
    status = $2,
    produced_qty = $3,
    updated_at = NOW()
WHERE code = $1
    RETURNING *;

-- name: UpdateScheduleTimes :one
UPDATE machine_schedules
SET
    start_time = $2,
    end_time = $3,
    updated_at = NOW()
WHERE code = $1
    RETURNING *;

-- name: DeleteSchedule :exec
UPDATE machine_schedules
SET is_active = FALSE, updated_at = NOW()
WHERE code = $1;
