-- name: CreateMachineType :one
INSERT INTO machine_types (
    code,
    name,
    description,
    type,
    requires_operator,
    is_active,
    created_by,
    enterprise_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, sqlc.arg(enterprise_id))
    RETURNING *;

-- name: UpdateMachineType :one
-- O WHERE usava $6, que é um parâmetro do SET: a alteração nunca casava a linha
-- certa. O código do tipo é parâmetro próprio.
UPDATE machine_types
SET
    name = sqlc.arg(name),
    description = sqlc.arg(description),
    type = sqlc.arg(type),
    requires_operator = sqlc.arg(requires_operator),
    is_active = sqlc.arg(is_active),
    updated_at = NOW()
WHERE code = sqlc.arg(code) AND enterprise_id = sqlc.arg(enterprise_id)
    RETURNING *;

-- name: GetMachineTypeByCode :one
SELECT *
FROM machine_types
WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: ListMachineTypes :many
SELECT *
FROM machine_types
WHERE is_active = TRUE AND enterprise_id = sqlc.arg(enterprise_id)
ORDER BY code;

-- name: DeleteMachineType :execrows
-- :execrows para o caso de uso saber se algo foi realmente inativado: excluir
-- um código inexistente devolvia 200 "sucesso".
UPDATE machine_types
SET is_active = FALSE, updated_at = NOW()
WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: CreateMachine :one
-- A tabela já tinha as colunas do cadastro completo (grupo, calendário, local,
-- criticidade, uso, aquisição, preparação, fornecedor, marca, preferencial e
-- responsável de manutenção); só o INSERT não as preenchia, e por isso nenhuma
-- delas podia ser informada.
INSERT INTO machines (
    code,
    name,
    machine_type_code,
    cost_center_code,
    capacity,
    capacity_unit,
    capacity_period,
    efficiency_rate,
    available_hours_per_day,
    is_active,
    resource_group_id,
    calendar_id,
    location,
    is_critical,
    usage_description,
    acquired_on,
    preparation_time,
    preparation_time_unit,
    supplier_code,
    brand,
    is_preferred,
    maintenance_responsible_employee_id,
    created_by,
    enterprise_id
)
VALUES (
    sqlc.arg(code), sqlc.arg(name), sqlc.arg(machine_type_code), sqlc.narg(cost_center_code),
    sqlc.arg(capacity), sqlc.arg(capacity_unit), sqlc.arg(capacity_period), sqlc.arg(efficiency_rate), sqlc.narg(available_hours_per_day),
    sqlc.arg(is_active), sqlc.narg(resource_group_id), sqlc.narg(calendar_id), sqlc.narg(location),
    sqlc.arg(is_critical), sqlc.narg(usage_description), sqlc.narg(acquired_on),
    sqlc.arg(preparation_time), sqlc.arg(preparation_time_unit), sqlc.narg(supplier_code),
    sqlc.narg(brand), sqlc.arg(is_preferred), sqlc.narg(maintenance_responsible_employee_id),
    sqlc.arg(created_by), sqlc.arg(enterprise_id))
    RETURNING *;

-- name: UpdateMachine :one
-- O WHERE usava $6 — que é `capacity_period`, não o código: a alteração
-- comparava um bigint com um enum e nunca encontrava a máquina.
UPDATE machines
SET
    name = sqlc.arg(name),
    machine_type_code = sqlc.arg(machine_type_code),
    cost_center_code = sqlc.narg(cost_center_code),
    capacity = sqlc.arg(capacity),
    capacity_unit = sqlc.arg(capacity_unit),
    capacity_period = sqlc.arg(capacity_period),
    efficiency_rate = sqlc.arg(efficiency_rate),
    available_hours_per_day = CASE WHEN sqlc.arg(inherit_work_center_hours)::boolean THEN NULL ELSE COALESCE(sqlc.narg(available_hours_per_day), available_hours_per_day) END,
    is_active = sqlc.arg(is_active),
    resource_group_id = sqlc.narg(resource_group_id),
    calendar_id = sqlc.narg(calendar_id),
    location = sqlc.narg(location),
    is_critical = sqlc.arg(is_critical),
    usage_description = sqlc.narg(usage_description),
    acquired_on = sqlc.narg(acquired_on),
    preparation_time = sqlc.arg(preparation_time),
    preparation_time_unit = sqlc.arg(preparation_time_unit),
    supplier_code = sqlc.narg(supplier_code),
    brand = sqlc.narg(brand),
    is_preferred = sqlc.arg(is_preferred),
    maintenance_responsible_employee_id = sqlc.narg(maintenance_responsible_employee_id),
    updated_at = NOW()
WHERE code = sqlc.arg(code) AND enterprise_id = sqlc.arg(enterprise_id)
    RETURNING *;

-- name: GetMachineByCode :one
SELECT *
FROM machines
WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: ListMachines :many
SELECT *
FROM machines
WHERE is_active = TRUE AND enterprise_id = sqlc.arg(enterprise_id)
ORDER BY code;

-- name: ListMachinesByType :many
SELECT *
FROM machines
WHERE machine_type_code = $1
  AND enterprise_id = sqlc.arg(enterprise_id)
  AND is_active = TRUE
ORDER BY code;

-- name: DeleteMachine :execrows
UPDATE machines
SET is_active = FALSE, updated_at = NOW()
WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: CreateItemMachineTime :one
INSERT INTO item_machine_times (
    item_code,
    mask,
    machine_code,
    production_time,
    production_time_unit,
    production_base_qty,
    setup_time,
    priority, efficiency_rate, time_basis,
    consumable_id, consumption_per_hour,
    enterprise_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, sqlc.narg(efficiency_rate), sqlc.arg(time_basis),
        sqlc.narg(consumable_id), sqlc.narg(consumption_per_hour), sqlc.arg(enterprise_id))
    ON CONFLICT (item_code, mask, machine_code)
DO UPDATE SET
    production_time = EXCLUDED.production_time,
    production_time_unit = EXCLUDED.production_time_unit,
    production_base_qty = EXCLUDED.production_base_qty,
    efficiency_rate = EXCLUDED.efficiency_rate,
    time_basis = EXCLUDED.time_basis,
    consumable_id = EXCLUDED.consumable_id,
    consumption_per_hour = EXCLUDED.consumption_per_hour,
    is_active = TRUE,
           setup_time = EXCLUDED.setup_time,
           priority = EXCLUDED.priority,
           updated_at = NOW()
           WHERE item_machine_times.enterprise_id = EXCLUDED.enterprise_id
           RETURNING *;


-- name: ListItemMachineTimes :many
SELECT *
FROM item_machine_times
WHERE (item_code = $1 OR $1 = 0)
  AND enterprise_id = sqlc.arg(enterprise_id)
  AND is_active = TRUE
ORDER BY item_code, machine_code, priority;

-- name: ListItemsByMachine :many
SELECT *
FROM item_machine_times
WHERE machine_code = $1
  AND enterprise_id = sqlc.arg(enterprise_id)
  AND is_active = TRUE
ORDER BY priority;

-- -- name: DeleteItemMachineTime :exec
-- UPDATE item_machine_times
-- SET is_active = FALSE, updated_at = NOW()
-- WHERE code = $1;

-- name: CreateSchedule :one
-- code is auto-assigned (MAX+1) so callers don't have to manage the business key;
-- the column is NOT NULL UNIQUE with no DB default.
--
-- Toda consulta desta tabela é filtrada por enterprise_id: sem isso, uma empresa
-- enxergava, alterava e excluía a fila de outra apenas informando o código.
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
    notes,
    enterprise_id
)
VALUES (
    COALESCE((SELECT MAX(code) FROM machine_schedules), 0) + 1,
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    RETURNING *;

-- name: GetSchedule :one
SELECT *
FROM machine_schedules
WHERE code = $1 AND enterprise_id = $2;

-- name: ListSchedules :many
SELECT *
FROM machine_schedules
WHERE machine_code = $1
  AND schedule_date = $2
  AND enterprise_id = $3
  AND is_active = TRUE
ORDER BY sequence;

-- name: ListSchedulesByRange :many
SELECT *
FROM machine_schedules
WHERE machine_code = $1
  AND schedule_date BETWEEN $2 AND $3
  AND enterprise_id = $4
  AND is_active = TRUE
ORDER BY schedule_date, sequence;

-- name: UpdateScheduleSequence :one
UPDATE machine_schedules
SET
    sequence = $2,
    priority_override = $3,
    updated_at = NOW()
WHERE code = $1 AND enterprise_id = $4
    RETURNING *;

-- name: UpdateScheduleStatus :one
UPDATE machine_schedules
SET
    status = $2,
    produced_qty = $3,
    updated_at = NOW()
WHERE code = $1 AND enterprise_id = $4
    RETURNING *;

-- name: UpdateScheduleTimes :one
UPDATE machine_schedules
SET
    start_time = $2,
    end_time = $3,
    updated_at = NOW()
WHERE code = $1 AND enterprise_id = $4
    RETURNING *;

-- name: DeleteSchedule :execrows
-- :execrows para o caso de uso saber se algo foi de fato removido: excluir um
-- slot inexistente devolvia 200 "sucesso".
UPDATE machine_schedules
SET is_active = FALSE, updated_at = NOW()
WHERE code = $1 AND enterprise_id = $2;


-- name: UpsertMachineConsumable :one
-- Consumível da máquina: quanto rende uma carga e quanto demora a troca. A TAXA
-- de consumo não mora aqui — ela depende do que está sendo produzido e fica em
-- item_machine_times.consumption_per_hour.
INSERT INTO machine_consumables (
    enterprise_id, machine_code, code, description, unit,
    capacity_per_refill, replacement_minutes
) VALUES (sqlc.arg(enterprise_id), $1, $2, $3, $4, $5, $6)
ON CONFLICT (enterprise_id, machine_code, code) DO UPDATE SET
    description = EXCLUDED.description,
    unit = EXCLUDED.unit,
    capacity_per_refill = EXCLUDED.capacity_per_refill,
    replacement_minutes = EXCLUDED.replacement_minutes,
    is_active = TRUE,
    updated_at = NOW()
RETURNING *;

-- name: ListMachineConsumables :many
SELECT * FROM machine_consumables
WHERE enterprise_id = sqlc.arg(enterprise_id)
  AND (machine_code = $1 OR $1 = 0)
  AND is_active = TRUE
ORDER BY machine_code, code;

-- name: DeleteMachineConsumable :execrows
UPDATE machine_consumables
SET is_active = FALSE, updated_at = NOW()
WHERE id = $1 AND enterprise_id = sqlc.arg(enterprise_id);
