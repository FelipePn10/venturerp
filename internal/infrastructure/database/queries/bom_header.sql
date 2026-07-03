-- name: CreateBomHeader :one
INSERT INTO bom_headers (item_code, mask, bom_type, version, status, valid_from, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetBomHeader :one
SELECT * FROM bom_headers WHERE id = $1;

-- name: ListBomHeadersByItem :many
SELECT * FROM bom_headers
WHERE item_code = $1 AND is_active = TRUE
ORDER BY version DESC;

-- name: UpdateBomHeaderStatus :one
UPDATE bom_headers SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: NextBomVersion :one
SELECT (COALESCE(MAX(version), 0) + 1)::INTEGER AS next_version
FROM bom_headers
WHERE item_code = $1 AND COALESCE(mask, '') = COALESCE($2, '');
