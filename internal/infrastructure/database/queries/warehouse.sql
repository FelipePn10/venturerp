-- name: ListWarehouses :many
SELECT
    id, code, description,
    location::text, type::text,
    disposition, reservations_allowed,
    created_by, created_at
FROM warehouse
ORDER BY id;

-- name: GetWarehouseByCode :one
SELECT
    id, code, description,
    location::text, type::text,
    disposition, reservations_allowed,
    created_by, created_at
FROM warehouse
WHERE code = $1;

-- name: CreateWarehouse :one
INSERT INTO warehouse (
    code,
    description,
    location,
    type,
    disposition,
    reservations_allowed,
    created_by
) VALUES (
    $1,
    $2,
    $3::warehouse_location,
    $4::warehouse_type,
    $5,
    $6,
    $7
) RETURNING
    id,
    code,
    description,
    location::text,
    type::text,
    disposition,
    reservations_allowed,
    created_by,
    created_at;
