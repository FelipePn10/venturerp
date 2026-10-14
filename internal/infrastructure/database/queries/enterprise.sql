-- name: CreateEnterprise :one
-- Quem cria a empresa fica vinculado a ela como ADMIN, na mesma instrução: sem
-- o vínculo, a empresa nasceria invisível para o próprio criador.
WITH nova AS (
    INSERT INTO enterprise (code, name, created_by, created_at)
    VALUES ($1, $2, $3, now())
    RETURNING *
), vinculo AS (
    INSERT INTO user_enterprises (user_id, enterprise_id, role)
    SELECT nova.created_by, nova.id, 'ADMIN' FROM nova WHERE nova.created_by IS NOT NULL
)
SELECT * FROM nova;

-- name: GetEnterpriseByID :one
SELECT * FROM enterprise WHERE id = $1;

-- name: GetEnterpriseByCodeForUser :one
-- Só devolve a empresa se o usuário estiver vinculado a ela.
SELECT e.* FROM enterprise e
  JOIN user_enterprises ue ON ue.enterprise_id = e.id
 WHERE e.code = $1 AND ue.user_id = $2;

-- name: ListEnterprisesForUser :many
SELECT e.* FROM enterprise e
  JOIN user_enterprises ue ON ue.enterprise_id = e.id
 WHERE ue.user_id = $1
 ORDER BY e.code;
