-- name: CreateIndependentDemand :one
INSERT INTO independent_demands (code, item_code, mask, cost_center_code, quantity, demand_date, created_by, enterprise_id,
                                 motivo, warehouse_id, customer_code, projeto, prioridade, observacao, programacao, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, @enterprise_id,
        @motivo, @warehouse_id, @customer_code, @projeto, @prioridade, @observacao, @programacao, $7)
    RETURNING *;

-- name: UpdateIndependentDemand :one
UPDATE independent_demands
SET item_code = $1,
    mask = $2,
    cost_center_code = $3,
    quantity = $4,
    demand_date = $5,
    motivo = @motivo,
    warehouse_id = @warehouse_id,
    customer_code = @customer_code,
    projeto = @projeto,
    prioridade = @prioridade,
    observacao = @observacao,
    updated_by = @updated_by,
    updated_at = NOW()
WHERE code = $6 AND enterprise_id = @enterprise_id AND situacao = 'ABERTA'
    RETURNING *;

-- name: NextIndependentDemandCode :one
-- O primeiro número livre da empresa: as demandas antigas geradas pelo pedido
-- de venda (pedido × 100000) não empurram a numeração para 200001.
SELECT COALESCE(MIN(d.code + 1), 1)::bigint
  FROM (SELECT i.code FROM independent_demands i WHERE i.enterprise_id = sqlc.arg(enterprise_id)::bigint AND i.code > 0
        UNION ALL SELECT 0::bigint) d
 WHERE NOT EXISTS (SELECT 1 FROM independent_demands x WHERE x.enterprise_id = sqlc.arg(enterprise_id)::bigint AND x.code = d.code + 1);

-- name: GetIndependentDemandByCode :one
SELECT * FROM independent_demands WHERE code = $1 AND enterprise_id = @enterprise_id;

-- name: ListIndependentDemands :many
SELECT * FROM independent_demands WHERE enterprise_id = @enterprise_id AND situacao = 'ABERTA' ORDER BY demand_date;

-- name: ListDemandsByItem :many
SELECT * FROM independent_demands WHERE item_code = $1 AND enterprise_id = @enterprise_id AND situacao = 'ABERTA' ORDER BY demand_date;

-- name: ListDemandsFromDate :many
SELECT * FROM independent_demands WHERE demand_date >= $1 AND enterprise_id = @enterprise_id AND situacao = 'ABERTA' ORDER BY demand_date;

-- name: DeleteIndependentDemand :exec
UPDATE independent_demands SET is_active = FALSE, situacao = 'CANCELADA', updated_at = NOW()
WHERE code = $1 AND enterprise_id = @enterprise_id;

-- name: CloseIndependentDemand :execrows
UPDATE independent_demands
   SET situacao = @situacao, is_active = FALSE,
       encerrada_em = NOW(), encerrada_por = @encerrada_por, motivo_encerramento = @motivo_encerramento,
       updated_by = @encerrada_por, updated_at = NOW()
 WHERE code = @code AND enterprise_id = @enterprise_id AND situacao = 'ABERTA';
