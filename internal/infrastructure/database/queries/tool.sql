-- Ferramenta é da empresa (enterprise_id). As consultas por id sem empresa são
-- usadas a partir de operação/roteiro já conferidos; o repositório confere a
-- posse antes (ver tool_serials.sql.go: ToolBelongsToEnterprise etc.).

-- ─── tools (master) ────────────────────────────────────────────────────────────

-- name: CreateTool :one
INSERT INTO tools (
    code, name, tool_type, life_type, life_limit, cost, status, created_by, enterprise_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateTool :one
UPDATE tools SET
    name       = $2,
    tool_type  = $3,
    life_type  = $4,
    life_limit = $5,
    cost       = $6,
    status     = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetTool :one
SELECT * FROM tools WHERE id = $1;

-- name: ListTools :many
SELECT * FROM tools
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE) AND enterprise_id = $2
ORDER BY code;

-- name: NextToolCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::BIGINT AS next_code FROM tools;

-- name: DeactivateTool :exec
UPDATE tools SET is_active = FALSE, updated_at = NOW() WHERE id = $1;

-- name: ConsumeToolLife :one
UPDATE tools SET life_used = life_used + $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ResetToolLife :one
UPDATE tools SET life_used = 0, status = 'ATIVA', updated_at = NOW()
WHERE id = $1
RETURNING *;

-- Ferramentas que já passaram do limite OU estão chegando nele.
--
-- Avisar só depois de estourar chega tarde: quem programa a semana precisa
-- saber que a matriz vence no meio dela para pedir a afiação antes, não depois
-- de a peça sair fora de medida. `sqlc.arg(threshold)` é a fração do limite a
-- partir da qual a ferramenta entra na lista (0.8 = 80 %); 1 devolve só as que
-- já estouraram, que é o comportamento antigo.
-- name: ListToolsNeedingReplacement :many
SELECT * FROM tools
WHERE is_active = TRUE AND life_limit > 0 AND enterprise_id = sqlc.arg(enterprise_id)
  AND life_used >= life_limit * sqlc.arg(threshold)::NUMERIC
ORDER BY (life_used / life_limit) DESC, code;

-- ─── route_operation_tools (association) ─────────────────────────────────────────

-- name: AddRouteOpTool :one
-- Etapa e ferramenta da empresa da sessão; senão, nada é gravado.
INSERT INTO route_operation_tools (route_operation_id, tool_id, qty_required)
SELECT $1, $2, $3
WHERE EXISTS (
      SELECT 1 FROM route_operations eo JOIN manufacturing_routes emr ON emr.id = eo.route_id
       WHERE eo.id = $1 AND emr.enterprise_id = sqlc.arg(enterprise_id))
  AND EXISTS (SELECT 1 FROM tools t WHERE t.id = $2 AND t.enterprise_id = sqlc.arg(enterprise_id))
RETURNING *;

-- name: RemoveRouteOpTool :execrows
DELETE FROM route_operation_tools
WHERE route_operation_tools.id = $1 AND EXISTS (
      SELECT 1 FROM route_operations eo JOIN manufacturing_routes emr ON emr.id = eo.route_id
       WHERE eo.id = route_operation_tools.route_operation_id AND emr.enterprise_id = sqlc.arg(enterprise_id));

-- name: ListToolsByRouteOp :many
SELECT rot.id, rot.route_operation_id, rot.tool_id, rot.qty_required,
       t.code AS tool_code, t.name AS tool_name, t.life_type, t.life_limit,
       t.life_used, t.status
FROM route_operation_tools rot
JOIN tools t ON t.id = rot.tool_id
WHERE rot.route_operation_id = $1 AND t.enterprise_id = sqlc.arg(enterprise_id)
ORDER BY rot.id;

-- name: ListToolsByRoute :many
SELECT rot.id, rot.route_operation_id, rot.tool_id, rot.qty_required,
       t.code AS tool_code, t.name AS tool_name, t.life_type, t.life_limit,
       t.life_used, t.status
FROM route_operation_tools rot
JOIN route_operations ro ON ro.id = rot.route_operation_id
JOIN manufacturing_routes mr ON mr.id = ro.route_id
JOIN tools t ON t.id = rot.tool_id
WHERE ro.route_id = $1 AND mr.enterprise_id = sqlc.arg(enterprise_id)
ORDER BY rot.route_operation_id, rot.id;
