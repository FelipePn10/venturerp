-- ─── operations ──────────────────────────────────────────────────────────────

-- name: CreateOperation :one
INSERT INTO operations (
    code, name, description, origin, situation,
    default_work_center_id, standard_time, setup_time,
    run_time, labor_time, run_time_base_qty,
    queue_time, wait_time, move_time, crew_size, time_unit,
    supplier_id, service_item_code, cost_per_unit, lead_time_days, third_party_remittance,
    scrap_pct, time_basis, time_basis_attribute,
    is_active, created_by, enterprise_id
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8,
    $9, $10, $11,
    $12, $13, $14, $15, $16,
    $17, $18, $19, $20, $21,
    $23, sqlc.arg(time_basis), sqlc.narg(time_basis_attribute),
    TRUE, $22, sqlc.arg(enterprise_id)
) RETURNING *;

-- name: UpdateOperation :one
UPDATE operations SET
    name = $2,
    description = $3,
    origin = $4,
    situation = $5,
    default_work_center_id = $6,
    standard_time = $7,
    setup_time = $8,
    run_time = $9,
    labor_time = $10,
    run_time_base_qty = $11,
    queue_time = $12,
    wait_time = $13,
    move_time = $14,
    crew_size = $15,
    time_unit = $16,
    supplier_id = $17,
    service_item_code = $18,
    cost_per_unit = $19,
    lead_time_days = $20,
    third_party_remittance = $21,
    scrap_pct = $22,
    time_basis = sqlc.arg(time_basis),
    time_basis_attribute = sqlc.narg(time_basis_attribute),
    updated_at = NOW()
WHERE id = $1 AND enterprise_id = sqlc.arg(enterprise_id)
RETURNING *;

-- name: GetOperationByID :one
SELECT * FROM operations WHERE id = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: ListOperations :many
SELECT * FROM operations
WHERE enterprise_id = sqlc.arg(enterprise_id)
  AND ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: DeactivateOperation :exec
UPDATE operations SET is_active = FALSE, updated_at = NOW() WHERE id = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: OperationUsedInRoutes :one
SELECT EXISTS(SELECT 1 FROM route_operations ro JOIN manufacturing_routes route ON route.id=ro.route_id WHERE ro.operation_id = $1 AND ro.is_active AND route.enterprise_id=sqlc.arg(enterprise_id));

-- name: NextOperationCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::BIGINT AS next_code FROM operations WHERE enterprise_id=sqlc.arg(enterprise_id);

-- ─── manufacturing_routes ─────────────────────────────────────────────────────

-- name: CreateRoute :one
INSERT INTO manufacturing_routes (
    code, item_code, mask, alternative, description,
    situation, is_standard, valid_from, valid_to, is_active, created_by, enterprise_id,
    is_template, lot_min, lot_max
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, TRUE, $10, sqlc.arg(enterprise_id),
    sqlc.arg(is_template), sqlc.narg(lot_min), sqlc.narg(lot_max)
) RETURNING *;

-- name: UpdateRoute :one
UPDATE manufacturing_routes SET
    description = $2,
    situation = $3,
    is_standard = $4,
    valid_from = $5,
    valid_to = $6,
    lot_min = sqlc.narg(lot_min),
    lot_max = sqlc.narg(lot_max),
    updated_at = NOW()
WHERE id = $1 AND enterprise_id = sqlc.arg(enterprise_id)
RETURNING *;

-- name: GetRouteByID :one
SELECT * FROM manufacturing_routes WHERE id = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: GetRouteByItemAndAlternative :one
SELECT * FROM manufacturing_routes
WHERE item_code = $1
  AND enterprise_id = sqlc.arg(enterprise_id)
  AND COALESCE(mask, '') = COALESCE($2, '')
  AND alternative = $3
  AND is_active = TRUE;

-- name: ListRoutesByItem :many
SELECT * FROM manufacturing_routes
WHERE item_code = $1 AND enterprise_id = sqlc.arg(enterprise_id) AND is_active = TRUE
ORDER BY alternative;

-- name: ListRoutesByItemWithInactive :many
SELECT * FROM manufacturing_routes
WHERE item_code = $1 AND enterprise_id = sqlc.arg(enterprise_id)
ORDER BY is_active DESC, alternative;

-- Reativa o roteiro. Se outro roteiro ativo do mesmo item/máscara já é o
-- padrão, o reativado volta como alternativo para não haver dois padrões.
-- name: ReactivateRoute :execrows
UPDATE manufacturing_routes mr
SET is_active = TRUE,
    is_standard = mr.is_standard AND NOT EXISTS (
        SELECT 1 FROM manufacturing_routes o
        WHERE o.enterprise_id = mr.enterprise_id AND o.item_code = mr.item_code
          AND COALESCE(o.mask, '') = COALESCE(mr.mask, '')
          AND o.id <> mr.id AND o.is_active AND o.is_standard),
    updated_at = NOW()
WHERE mr.id = $1 AND mr.enterprise_id = sqlc.arg(enterprise_id) AND NOT mr.is_active;

-- name: ClearOverlappingStandardRoutes :exec
-- Um item/máscara tem UM roteiro padrão por período. Gravar um padrão tira o
-- "padrão" dos outros cuja vigência cruza com a dele — padrões em janelas
-- diferentes (um até junho, outro a partir de julho) continuam valendo.
UPDATE manufacturing_routes o SET is_standard = FALSE, updated_at = NOW()
FROM manufacturing_routes r
WHERE r.id = $1 AND r.enterprise_id = sqlc.arg(enterprise_id)
  AND o.enterprise_id = r.enterprise_id
  AND o.id <> r.id
  AND o.item_code IS NOT DISTINCT FROM r.item_code
  AND COALESCE(o.mask, '') = COALESCE(r.mask, '')
  AND o.is_active AND o.is_standard
  AND COALESCE(o.valid_from, DATE '0001-01-01') <= COALESCE(r.valid_to, DATE '9999-12-31')
  AND COALESCE(r.valid_from, DATE '0001-01-01') <= COALESCE(o.valid_to, DATE '9999-12-31');

-- name: SyncPrimaryResourceWithStep :exec
-- O centro da etapa É o recurso principal. Quando a etapa muda de centro, o
-- recurso principal acompanha: o do novo centro passa a ser o principal (se
-- estiver na lista de alternativos) e nenhum outro fica marcado. Antes a etapa
-- dizia um centro e a lista de recursos dizia outro.
UPDATE route_operation_resources r
SET is_primary = (r.work_center_id IS NOT DISTINCT FROM step.work_center_id), updated_at = NOW()
FROM route_operations step
JOIN manufacturing_routes mr ON mr.id = step.route_id
WHERE step.id = $1 AND r.route_operation_id = step.id
  AND mr.enterprise_id = sqlc.arg(enterprise_id)
  AND r.is_primary <> (r.work_center_id IS NOT DISTINCT FROM step.work_center_id);

-- name: DeactivateRoute :exec
UPDATE manufacturing_routes SET is_active = FALSE, updated_at = NOW() WHERE id = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: NextRouteCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::BIGINT AS next_code FROM manufacturing_routes WHERE enterprise_id=sqlc.arg(enterprise_id);

-- name: ItemHasRoute :one
SELECT EXISTS(
    SELECT 1 FROM manufacturing_routes
    WHERE item_code = $1 AND enterprise_id=sqlc.arg(enterprise_id) AND is_active = TRUE
) AS has_route;

-- name: GetStandardRouteForItem :one
-- A escolha mora numa função só (erp_roteiro_vigente, migrações 476/477), que
-- OP, MRP, ordem de serviço de terceiro, liberação e rateio usam:
--   máscara exata vence o genérico; faixa de lote que casa vence o roteiro sem
--   faixa; depois padrão, vigência mais recente, menor alternativa.
-- Quantidade nula ignora as faixas (prazo do item por unidade, telas).
SELECT * FROM manufacturing_routes
WHERE id = erp_roteiro_vigente(sqlc.arg(enterprise_id)::bigint, sqlc.arg(item_code)::bigint,
                               sqlc.narg(mask)::text, sqlc.narg(ref_date)::date, sqlc.narg(quantity)::numeric);

-- name: ListRouteTemplates :many
SELECT * FROM manufacturing_routes
WHERE is_template AND enterprise_id = sqlc.arg(enterprise_id)
  AND (sqlc.arg(include_inactive)::boolean OR is_active)
ORDER BY is_active DESC, description, code;

-- ─── route_operations ────────────────────────────────────────────────────────

-- name: AddRouteOperation :one
-- Só entra em roteiro da empresa, com operação e centro de trabalho dela;
-- fora disso não grava nada (sem linha = não encontrado).
INSERT INTO route_operations (
    route_id, sequence, operation_id, work_center_id,
    standard_time, setup_time,
    run_time, labor_time, run_time_base_qty,
    queue_time, wait_time, move_time, crew_size, time_unit,
    supplier_id, service_item_code, cost_per_unit, lead_time_days, third_party_remittance,
    scrap_pct, inspection_required,
    situation, notes, is_active,
    time_basis, time_basis_attribute
)
SELECT
    $1, $2, $3, $4,
    $5, $6,
    $7, $8, $9,
    $10, $11, $12, $13, $14,
    $15, $16, $17, $18, $19,
    $22, $23,
    $20, $21, TRUE,
    sqlc.narg(time_basis), sqlc.narg(time_basis_attribute)
WHERE EXISTS (SELECT 1 FROM manufacturing_routes mr WHERE mr.id = $1 AND mr.enterprise_id = sqlc.arg(enterprise_id))
  AND EXISTS (SELECT 1 FROM operations op WHERE op.id = $3 AND op.enterprise_id = sqlc.arg(enterprise_id))
  AND ($4::bigint IS NULL OR EXISTS (SELECT 1 FROM machine_types mt WHERE mt.id = $4 AND mt.enterprise_id = sqlc.arg(enterprise_id)))
RETURNING *;

-- name: UpdateRouteOperation :one
UPDATE route_operations SET
    work_center_id = $2,
    standard_time = $3,
    setup_time = $4,
    run_time = $5,
    labor_time = $6,
    run_time_base_qty = $7,
    queue_time = $8,
    wait_time = $9,
    move_time = $10,
    crew_size = $11,
    time_unit = $12,
    supplier_id = $13,
    service_item_code = $14,
    cost_per_unit = $15,
    lead_time_days = $16,
    third_party_remittance = $17,
    scrap_pct = $20,
    inspection_required = $21,
    situation = $18,
    notes = $19,
    time_basis = sqlc.narg(time_basis),
    time_basis_attribute = sqlc.narg(time_basis_attribute),
    updated_at = NOW()
WHERE route_operations.id = $1
  AND EXISTS (
      SELECT 1 FROM route_operations eo JOIN manufacturing_routes emr ON emr.id = eo.route_id
       WHERE eo.id = route_operations.id AND emr.enterprise_id = sqlc.arg(enterprise_id))
RETURNING *;
-- O centro da etapa é conferido antes, na mesma transação
-- (WorkCenterOfEnterprise): repetir o parâmetro do centro no WHERE é o padrão que
-- TestWhereNaoReusaParametroDoSet proíbe.

-- name: WorkCenterOfEnterprise :one
SELECT EXISTS (SELECT 1 FROM machine_types mt WHERE mt.id = $1 AND mt.enterprise_id = sqlc.arg(enterprise_id));

-- name: GetRouteOperations :many
SELECT
    ro.*,
    op.name AS operation_name,
    op.origin AS operation_origin,
    op.standard_time AS op_standard_time,
    op.setup_time AS op_setup_time,
    op.run_time AS op_run_time,
    op.labor_time AS op_labor_time,
    op.run_time_base_qty AS op_run_time_base_qty,
    op.queue_time AS op_queue_time,
    op.wait_time AS op_wait_time,
    op.move_time AS op_move_time,
    op.crew_size AS op_crew_size,
    op.time_unit AS op_time_unit,
    op.scrap_pct AS op_scrap_pct,
    op.time_basis AS op_time_basis,
    op.time_basis_attribute AS op_time_basis_attribute,
    COALESCE(item.engineering_dimensions, '{}'::jsonb) AS item_dimensions,
    COALESCE(item.pdm_attributes, '[]'::jsonb) AS item_pdm_attributes,
    COALESCE(ro.lead_time_days, op.lead_time_days, 0)::int AS effective_lead_time_days,
    COALESCE(ro.cost_per_unit, op.cost_per_unit) AS effective_cost_per_unit,
    COALESCE(ro.supplier_id, op.supplier_id) AS effective_supplier_id,
    COALESCE(ro.work_center_id, op.default_work_center_id) AS effective_work_center_id,
    mt.name AS work_center_name,
    COALESCE(mt.requires_operator, TRUE) AS requires_operator
FROM route_operations ro
JOIN manufacturing_routes mr ON mr.id = ro.route_id
JOIN operations op ON op.id = ro.operation_id
LEFT JOIN machine_types mt ON mt.id = COALESCE(ro.work_center_id, op.default_work_center_id)
LEFT JOIN items item ON item.code = mr.item_code
WHERE ro.route_id = $1 AND ro.is_active = TRUE
  AND mr.enterprise_id = sqlc.arg(enterprise_id)
ORDER BY ro.sequence;

-- name: RemoveRouteOperation :execrows
-- Tira a etapa do roteiro sem partir a rede. Numa instrução só (atômica):
--   1. quem vinha antes dela passa a ligar direto em quem vinha depois
--      (A→B→C sem B vira A→C, e não "C livre para começar junto com A");
--   2. as dependências da etapa são apagadas;
--   3. a etapa é desativada.
-- Antes as dependências ficavam apontando para a etapa desativada, o caminho
-- crítico via uma predecessora que não existia mais e acusava CICLO — a OP do
-- item passava a ser recusada com "o roteiro contém um ciclo de dependências".
WITH alvo AS (
    SELECT eo.id FROM route_operations eo
    JOIN manufacturing_routes emr ON emr.id = eo.route_id
    WHERE eo.id = $1 AND eo.is_active AND emr.enterprise_id = sqlc.arg(enterprise_id)
), ponte AS (
    INSERT INTO route_operation_network (predecessor_id, successor_id, overlap_pct)
    SELECT antes.predecessor_id, depois.successor_id, 0
    FROM route_operation_network antes
    JOIN route_operation_network depois ON depois.predecessor_id = antes.successor_id
    WHERE antes.successor_id IN (SELECT id FROM alvo)
      AND antes.predecessor_id <> depois.successor_id
    ON CONFLICT (predecessor_id, successor_id) DO NOTHING
    RETURNING 1
), apaga AS (
    DELETE FROM route_operation_network
    WHERE predecessor_id IN (SELECT id FROM alvo) OR successor_id IN (SELECT id FROM alvo)
    RETURNING 1
)
UPDATE route_operations SET is_active = FALSE, updated_at = NOW()
WHERE route_operations.id IN (SELECT id FROM alvo);

-- ─── route_operation_network ─────────────────────────────────────────────────

-- name: UpsertNetworkEdge :one
-- As duas etapas precisam ser da empresa (e do mesmo roteiro).
INSERT INTO route_operation_network (predecessor_id, successor_id, overlap_pct)
SELECT $1, $2, $3
WHERE EXISTS (
      SELECT 1 FROM route_operations eo JOIN manufacturing_routes emr ON emr.id = eo.route_id
       WHERE eo.id = $1 AND emr.enterprise_id = sqlc.arg(enterprise_id))
  AND EXISTS (SELECT 1 FROM route_operations a JOIN route_operations b ON b.route_id = a.route_id
               WHERE a.id = $1 AND b.id = $2)
ON CONFLICT (predecessor_id, successor_id)
DO UPDATE SET overlap_pct = EXCLUDED.overlap_pct
RETURNING *;

-- name: DeleteNetworkEdge :execrows
DELETE FROM route_operation_network
WHERE predecessor_id = $1 AND successor_id = $2
  AND EXISTS (
      SELECT 1 FROM route_operations eo JOIN manufacturing_routes emr ON emr.id = eo.route_id
       WHERE eo.id = route_operation_network.predecessor_id AND emr.enterprise_id = sqlc.arg(enterprise_id));

-- name: GetNetworkEdges :many
-- As duas pontas precisam estar ativas: uma dependência que sobrou de etapa
-- removida apontaria para um nó que o caminho crítico não conhece.
SELECT ron.*
FROM route_operation_network ron
JOIN route_operations ro ON ro.id = ron.predecessor_id AND ro.is_active
JOIN route_operations sucessora ON sucessora.id = ron.successor_id AND sucessora.is_active
JOIN manufacturing_routes mr ON mr.id = ro.route_id
WHERE ro.route_id = $1 AND mr.enterprise_id = sqlc.arg(enterprise_id)
ORDER BY ron.predecessor_id, ron.successor_id;

-- name: GetExternalRouteOpsForRoute :many
-- Etapas externas do roteiro JÁ ESCOLHIDO (GetStandardRouteForItem). Antes a
-- consulta escolhia sozinha — `is_standard` sem vigência, sem máscara e sem
-- empresa — e o MRP gerava a ordem de serviço de um roteiro diferente do que a
-- OP explodia. O fornecedor também era procurado por id OU código em todas as
-- empresas; agora só na empresa do roteiro.
SELECT
    ro.id,
    ro.operation_id,
    ro.work_center_id,
    op.name AS operation_name,
    op.origin,
    COALESCE(ro.run_time, ro.standard_time, NULLIF(op.run_time, 0), op.standard_time) AS effective_hours,
    COALESCE((SELECT supplier.code FROM suppliers supplier
              WHERE supplier.enterprise_id = mr.enterprise_id
                AND (supplier.id = COALESCE(ro.supplier_id, op.supplier_id)
                     OR supplier.code = COALESCE(ro.supplier_id, op.supplier_id))
              -- A tela grava o CÓDIGO do fornecedor (loadSuppliers): o código vence;
              -- o id fica só para vínculo antigo gravado por id.
              ORDER BY (supplier.code = COALESCE(ro.supplier_id, op.supplier_id)) DESC
              LIMIT 1), 0)::bigint AS supplier_id,
    COALESCE(ro.service_item_code, op.service_item_code) AS service_item_code,
    COALESCE(ro.cost_per_unit, op.cost_per_unit, 0) AS cost_per_unit,
    COALESCE(ro.lead_time_days, op.lead_time_days, 0) AS lead_time_days,
    COALESCE(ro.third_party_remittance, op.third_party_remittance, 'DEMAND_ITEMS') AS remittance_type
FROM manufacturing_routes mr
JOIN route_operations ro ON ro.route_id = mr.id
JOIN operations op ON op.id = ro.operation_id AND op.enterprise_id = mr.enterprise_id
WHERE mr.id = $1
  AND mr.enterprise_id = sqlc.arg(enterprise_id)
  AND ro.is_active = TRUE
  AND ro.situation NOT IN ('INATIVA', 'FANTASMA')
  AND op.origin IN ('EXTERNA', 'TERCEIROS')
ORDER BY ro.sequence;

-- ─── route_operation_resources (alternative work centers) ─────────────────────

-- name: AddRouteOpResource :one
INSERT INTO route_operation_resources (
    route_operation_id, work_center_id, priority, time_factor, is_primary
)
SELECT $1, $2, $3, $4, $5
WHERE EXISTS (
      SELECT 1 FROM route_operations eo JOIN manufacturing_routes emr ON emr.id = eo.route_id
       WHERE eo.id = $1 AND emr.enterprise_id = sqlc.arg(enterprise_id))
  AND EXISTS (SELECT 1 FROM machine_types mt WHERE mt.id = $2 AND mt.enterprise_id = sqlc.arg(enterprise_id))
RETURNING *;

-- name: UpdateRouteOpResource :one
UPDATE route_operation_resources SET
    priority    = $2,
    time_factor = $3,
    updated_at  = NOW()
WHERE route_operation_resources.id = $1 AND EXISTS (
      SELECT 1 FROM route_operations eo JOIN manufacturing_routes emr ON emr.id = eo.route_id
       WHERE eo.id = route_operation_resources.route_operation_id AND emr.enterprise_id = sqlc.arg(enterprise_id))
RETURNING *;

-- name: GetRouteOpResource :one
SELECT * FROM route_operation_resources
WHERE route_operation_resources.id = $1 AND EXISTS (
      SELECT 1 FROM route_operations eo JOIN manufacturing_routes emr ON emr.id = eo.route_id
       WHERE eo.id = route_operation_resources.route_operation_id AND emr.enterprise_id = sqlc.arg(enterprise_id));

-- name: RemoveRouteOpResource :execrows
DELETE FROM route_operation_resources
WHERE route_operation_resources.id = $1 AND EXISTS (
      SELECT 1 FROM route_operations eo JOIN manufacturing_routes emr ON emr.id = eo.route_id
       WHERE eo.id = route_operation_resources.route_operation_id AND emr.enterprise_id = sqlc.arg(enterprise_id));

-- name: ClearPrimaryResources :exec
UPDATE route_operation_resources SET is_primary = FALSE, updated_at = NOW()
WHERE route_operation_id = $1 AND EXISTS (
      SELECT 1 FROM route_operations eo JOIN manufacturing_routes emr ON emr.id = eo.route_id
       WHERE eo.id = $1 AND emr.enterprise_id = sqlc.arg(enterprise_id));

-- name: KeepStepCenterAsAlternative :exec
-- Antes de outro centro virar o principal, o centro atual da etapa (o dela ou o
-- herdado da operação) entra na lista de alternativos, com a prioridade que o
-- promovido tinha. Sem isso o centro antigo sumia da etapa na troca.
INSERT INTO route_operation_resources (route_operation_id, work_center_id, priority, time_factor, is_primary)
SELECT ro.id, COALESCE(ro.work_center_id, op.default_work_center_id), promovido.priority, 1, FALSE
FROM route_operations ro
JOIN operations op ON op.id = ro.operation_id
JOIN manufacturing_routes mr ON mr.id = ro.route_id
JOIN route_operation_resources promovido ON promovido.id = sqlc.arg(resource_id) AND promovido.route_operation_id = ro.id
WHERE ro.id = sqlc.arg(route_operation_id) AND mr.enterprise_id = sqlc.arg(enterprise_id)
  AND COALESCE(ro.work_center_id, op.default_work_center_id) IS NOT NULL
  AND COALESCE(ro.work_center_id, op.default_work_center_id) <> promovido.work_center_id
ON CONFLICT DO NOTHING;

-- name: SetResourcePrimary :one
UPDATE route_operation_resources SET is_primary = TRUE, updated_at = NOW()
WHERE route_operation_resources.id = $1 AND EXISTS (
      SELECT 1 FROM route_operations eo JOIN manufacturing_routes emr ON emr.id = eo.route_id
       WHERE eo.id = route_operation_resources.route_operation_id AND emr.enterprise_id = sqlc.arg(enterprise_id))
RETURNING *;

-- name: SetRouteOpWorkCenter :exec
UPDATE route_operations SET work_center_id = $2, updated_at = NOW()
WHERE route_operations.id = $1 AND EXISTS (
      SELECT 1 FROM route_operations eo JOIN manufacturing_routes emr ON emr.id = eo.route_id
       WHERE eo.id = route_operations.id AND emr.enterprise_id = sqlc.arg(enterprise_id));

-- name: ListResourcesByRouteOp :many
SELECT r.*, mt.name AS work_center_name
FROM route_operation_resources r
LEFT JOIN machine_types mt ON mt.id = r.work_center_id
WHERE r.route_operation_id = $1 AND EXISTS (
      SELECT 1 FROM route_operations eo JOIN manufacturing_routes emr ON emr.id = eo.route_id
       WHERE eo.id = r.route_operation_id AND emr.enterprise_id = sqlc.arg(enterprise_id))
ORDER BY r.is_primary DESC, r.priority, r.id;

-- name: ListResourcesByRoute :many
SELECT r.*, mt.name AS work_center_name
FROM route_operation_resources r
JOIN route_operations ro ON ro.id = r.route_operation_id
JOIN manufacturing_routes mr ON mr.id = ro.route_id
LEFT JOIN machine_types mt ON mt.id = r.work_center_id
WHERE ro.route_id = $1 AND mr.enterprise_id = sqlc.arg(enterprise_id)
ORDER BY r.route_operation_id, r.is_primary DESC, r.priority, r.id;

-- ─── operation_documents (desenho / instrução / ficha de processo) ────────────

-- name: CreateOperationDocument :one
INSERT INTO operation_documents (
    operation_id, route_operation_id, kind, title, reference, revision, instructions,
    is_active, enterprise_id, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, sqlc.arg(enterprise_id), $8)
RETURNING *;

-- name: UpdateOperationDocument :one
UPDATE operation_documents SET
    kind = $2, title = $3, reference = $4, revision = $5, instructions = $6,
    updated_at = NOW()
WHERE id = $1 AND enterprise_id = sqlc.arg(enterprise_id)
RETURNING *;

-- name: DeactivateOperationDocument :exec
UPDATE operation_documents SET is_active = FALSE, updated_at = NOW()
WHERE id = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: ListDocumentsByOperation :many
SELECT * FROM operation_documents
WHERE operation_id = $1 AND is_active = TRUE AND enterprise_id = sqlc.arg(enterprise_id)
ORDER BY kind, title;

-- Documentos que o operador vê numa etapa: os da própria etapa MAIS os que a
-- operação de biblioteca carrega. A instrução genérica ("como rebarbar") vale em
-- todo roteiro; o desenho é desta etapa. Separar os dois na leitura obrigaria a
-- tela a fazer duas chamadas e juntar — e alguém esqueceria uma delas.
-- name: ListDocumentsForRouteOperation :many
SELECT d.*, (d.route_operation_id IS NOT NULL) AS is_step_level
FROM operation_documents d
WHERE d.is_active = TRUE
  AND d.enterprise_id = sqlc.arg(enterprise_id)
  AND (
        d.route_operation_id = sqlc.arg(route_operation_id)
     OR d.operation_id = (SELECT ro.operation_id FROM route_operations ro
                          WHERE ro.id = sqlc.arg(route_operation_id))
  )
ORDER BY is_step_level DESC, d.kind, d.title;

-- name: ListDocumentsByRoute :many
SELECT d.*, (d.route_operation_id IS NOT NULL) AS is_step_level, ro.id AS step_id, ro.sequence
FROM route_operations ro
JOIN manufacturing_routes mr ON mr.id = ro.route_id
JOIN operation_documents d
  ON d.route_operation_id = ro.id OR d.operation_id = ro.operation_id
WHERE ro.route_id = $1 AND ro.is_active = TRUE
  AND d.is_active = TRUE
  AND mr.enterprise_id = sqlc.arg(enterprise_id)
  AND d.enterprise_id = sqlc.arg(enterprise_id)
ORDER BY ro.sequence, is_step_level DESC, d.kind, d.title;

-- ─── ponto de inspeção no roteiro ────────────────────────────────────────────

-- name: ListInspectionPlansByRoute :many
SELECT ip.*, ro.sequence AS step_sequence,
       (SELECT count(*) FROM inspection_plan_characteristics c WHERE c.plan_id = ip.id) AS characteristic_count
FROM inspection_plans ip
JOIN route_operations ro ON ro.id = ip.route_operation_id
JOIN manufacturing_routes mr ON mr.id = ro.route_id
WHERE ro.route_id = $1 AND ro.is_active = TRUE AND ip.is_active = TRUE
  AND mr.enterprise_id = sqlc.arg(enterprise_id)
ORDER BY ro.sequence, ip.id;

-- Etapas marcadas como inspecionadas na ordem: é daqui que nasce o registro de
-- inspeção quando a ordem de produção é criada.
-- name: ListInspectionStepsForOrderRoute :many
SELECT ro.id AS route_operation_id, ro.sequence, ip.id AS plan_id, ip.item_code
FROM route_operations ro
JOIN manufacturing_routes mr ON mr.id = ro.route_id
LEFT JOIN inspection_plans ip
       ON ip.route_operation_id = ro.id AND ip.is_active = TRUE
WHERE ro.route_id = $1
  AND ro.is_active = TRUE
  AND ro.inspection_required = TRUE
  AND mr.enterprise_id = sqlc.arg(enterprise_id)
ORDER BY ro.sequence;

-- Descobre a que roteiro uma etapa pertence. O filtro por empresa vive aqui,
-- não no chamador: a etapa é endereçada por id e sem isto seria possível
-- alcançar a etapa de outra empresa informando o número.
-- name: RouteIDOfOperation :one
SELECT ro.route_id
FROM route_operations ro
JOIN manufacturing_routes mr ON mr.id = ro.route_id
WHERE ro.id = $1 AND mr.enterprise_id = sqlc.arg(enterprise_id);

-- Etapas de uma ORDEM que são ponto de inspeção e continuam sem plano ativo.
--
-- A pendência não é do instante em que a ordem foi criada: ela dura até alguém
-- cadastrar o plano. Por isso a conferência vive na LISTAGEM das etapas, e não
-- só na explosão do roteiro — senão o aviso aparecia uma vez e sumia no primeiro
-- recarregamento da tela.
-- name: ListOrderStepsMissingInspectionPlan :many
SELECT poo.sequence
FROM production_order_operations poo
JOIN production_orders po ON po.id = poo.production_order_id
JOIN route_operations ro ON ro.id = poo.route_operation_id
LEFT JOIN inspection_plans ip ON ip.route_operation_id = ro.id AND ip.is_active = TRUE
WHERE poo.production_order_id = sqlc.arg(production_order_id)
  AND po.enterprise_id = sqlc.arg(enterprise_id)
  AND ro.inspection_required = TRUE
  AND ip.id IS NULL
ORDER BY poo.sequence;

-- ─── cópia de roteiro (uma transação, ver RoutingRepositorySQLC.CopyRoute) ──
-- A correspondência etapa de origem → etapa nova é pela SEQUÊNCIA, que é única
-- entre as etapas ativas de um roteiro.

-- name: CopyRouteSteps :execrows
INSERT INTO route_operations (
    route_id, sequence, operation_id, work_center_id,
    standard_time, setup_time, run_time, labor_time, run_time_base_qty,
    queue_time, wait_time, move_time, crew_size, time_unit,
    supplier_id, service_item_code, cost_per_unit, lead_time_days, third_party_remittance,
    scrap_pct, inspection_required, situation, notes, is_active,
    time_basis, time_basis_attribute
)
SELECT sqlc.arg(target_route_id), ro.sequence, ro.operation_id, ro.work_center_id,
    ro.standard_time, ro.setup_time, ro.run_time, ro.labor_time, ro.run_time_base_qty,
    ro.queue_time, ro.wait_time, ro.move_time, ro.crew_size, ro.time_unit,
    ro.supplier_id, ro.service_item_code, ro.cost_per_unit, ro.lead_time_days, ro.third_party_remittance,
    ro.scrap_pct, ro.inspection_required, ro.situation, ro.notes, TRUE,
    ro.time_basis, ro.time_basis_attribute
FROM route_operations ro
JOIN manufacturing_routes origem ON origem.id = ro.route_id
JOIN manufacturing_routes destino ON destino.id = sqlc.arg(target_route_id)
WHERE ro.route_id = sqlc.arg(source_route_id) AND ro.is_active
  AND origem.enterprise_id = sqlc.arg(enterprise_id)
  AND destino.enterprise_id = sqlc.arg(enterprise_id)
ORDER BY ro.sequence;

-- name: CopyRouteNetwork :execrows
INSERT INTO route_operation_network (predecessor_id, successor_id, overlap_pct)
SELECT np.id, ns.id, e.overlap_pct
FROM route_operation_network e
JOIN route_operations sp ON sp.id = e.predecessor_id AND sp.is_active AND sp.route_id = sqlc.arg(source_route_id)
JOIN route_operations ss ON ss.id = e.successor_id AND ss.is_active
JOIN route_operations np ON np.route_id = sqlc.arg(target_route_id) AND np.is_active AND np.sequence = sp.sequence
JOIN route_operations ns ON ns.route_id = sqlc.arg(target_route_id) AND ns.is_active AND ns.sequence = ss.sequence
JOIN manufacturing_routes origem ON origem.id = sp.route_id AND origem.enterprise_id = sqlc.arg(enterprise_id)
ON CONFLICT (predecessor_id, successor_id) DO NOTHING;

-- name: CopyRouteResources :execrows
INSERT INTO route_operation_resources (route_operation_id, work_center_id, priority, time_factor, is_primary)
SELECT novo.id, r.work_center_id, r.priority, r.time_factor, r.is_primary
FROM route_operation_resources r
JOIN route_operations velho ON velho.id = r.route_operation_id AND velho.is_active AND velho.route_id = sqlc.arg(source_route_id)
JOIN route_operations novo ON novo.route_id = sqlc.arg(target_route_id) AND novo.is_active AND novo.sequence = velho.sequence
JOIN manufacturing_routes origem ON origem.id = velho.route_id AND origem.enterprise_id = sqlc.arg(enterprise_id)
ON CONFLICT DO NOTHING;

-- name: CopyRouteTools :execrows
INSERT INTO route_operation_tools (route_operation_id, tool_id, qty_required)
SELECT novo.id, t.tool_id, t.qty_required
FROM route_operation_tools t
JOIN route_operations velho ON velho.id = t.route_operation_id AND velho.is_active AND velho.route_id = sqlc.arg(source_route_id)
JOIN route_operations novo ON novo.route_id = sqlc.arg(target_route_id) AND novo.is_active AND novo.sequence = velho.sequence
JOIN manufacturing_routes origem ON origem.id = velho.route_id AND origem.enterprise_id = sqlc.arg(enterprise_id)
ON CONFLICT DO NOTHING;

-- name: CopyRouteStepDocuments :execrows
INSERT INTO operation_documents (route_operation_id, kind, title, reference, revision, instructions, is_active, enterprise_id, created_by)
SELECT novo.id, d.kind, d.title, d.reference, d.revision, d.instructions, TRUE, d.enterprise_id, sqlc.arg(created_by)
FROM operation_documents d
JOIN route_operations velho ON velho.id = d.route_operation_id AND velho.is_active AND velho.route_id = sqlc.arg(source_route_id)
JOIN route_operations novo ON novo.route_id = sqlc.arg(target_route_id) AND novo.is_active AND novo.sequence = velho.sequence
WHERE d.is_active AND d.enterprise_id = sqlc.arg(enterprise_id);

-- name: CopyRouteInspectionPlan :one
-- Um plano por chamada: as características vão em seguida, apontando para o
-- plano novo que esta instrução devolve.
INSERT INTO inspection_plans (item_code, route_operation_id, point_type, description, sample_size,
                              acceptance_level, instructions, is_active, created_by, sampling_procedure_id)
SELECT sqlc.arg(target_item_code), novo.id, p.point_type, p.description, p.sample_size,
       p.acceptance_level, p.instructions, TRUE, sqlc.arg(created_by), p.sampling_procedure_id
FROM inspection_plans p
JOIN route_operations velho ON velho.id = p.route_operation_id AND velho.is_active
JOIN manufacturing_routes origem ON origem.id = velho.route_id AND origem.enterprise_id = sqlc.arg(enterprise_id)
JOIN route_operations novo ON novo.route_id = sqlc.arg(target_route_id) AND novo.is_active AND novo.sequence = velho.sequence
WHERE p.id = sqlc.arg(source_plan_id) AND p.is_active
RETURNING id;

-- name: CopyInspectionCharacteristics :execrows
INSERT INTO inspection_plan_characteristics (plan_id, name, nominal, tolerance_upper, tolerance_lower, unit, is_critical)
SELECT sqlc.arg(target_plan_id), c.name, c.nominal, c.tolerance_upper, c.tolerance_lower, c.unit, c.is_critical
FROM inspection_plan_characteristics c
WHERE c.plan_id = sqlc.arg(source_plan_id)
ORDER BY c.id;

-- name: ListActiveInspectionPlanIDsByRoute :many
SELECT p.id
FROM inspection_plans p
JOIN route_operations ro ON ro.id = p.route_operation_id AND ro.is_active
JOIN manufacturing_routes mr ON mr.id = ro.route_id
WHERE ro.route_id = $1 AND p.is_active AND mr.enterprise_id = sqlc.arg(enterprise_id)
ORDER BY ro.sequence, p.id;

-- ─── componentes por etapa ───────────────────────────────────────────────────

-- name: ListRouteMaterials :many
-- Componentes da estrutura do item do roteiro (os que valem para a máscara do
-- roteiro) e a etapa em que cada um entra.
SELECT s.id AS structure_id, s.child_code, i.business_code AS child_business_code,
       COALESCE(i.name, '')::text AS child_name, s.quantity, s.unit_of_measurement,
       s.route_operation_sequence, s.parent_mask, s.is_coproduct
FROM item_structures s
JOIN manufacturing_routes mr ON mr.item_code = s.parent_code
JOIN items i ON i.code = s.child_code
WHERE mr.id = $1 AND mr.enterprise_id = sqlc.arg(enterprise_id) AND s.is_active
  AND (s.parent_mask IS NULL OR s.parent_mask = '' OR COALESCE(mr.mask, '') = '' OR s.parent_mask = mr.mask)
ORDER BY s.sequence, s.id;

-- name: SetStructureRouteStep :execrows
UPDATE item_structures s SET route_operation_sequence = sqlc.narg(step_sequence), updated_at = NOW()
FROM manufacturing_routes mr
WHERE s.id = sqlc.arg(structure_id) AND mr.id = sqlc.arg(route_id)
  AND mr.enterprise_id = sqlc.arg(enterprise_id) AND s.parent_code = mr.item_code AND s.is_active;

-- ─── tempo padrão × real apontado ────────────────────────────────────────────

-- name: RouteStepActuals :many
-- Por etapa do roteiro: o que as OPs concluídas no período planejaram e o que
-- de fato apontaram. Só etapa concluída e com hora apontada entra.
SELECT poo.route_operation_id,
       count(*)::int AS orders,
       COALESCE(sum(poo.actual_hours), 0)::numeric AS actual_hours,
       COALESCE(sum(poo.planned_hours), 0)::numeric AS planned_hours,
       COALESCE(sum(poo.setup_hours), 0)::numeric AS setup_hours,
       COALESCE(sum(poo.planned_qty), 0)::numeric AS quantity,
       max(poo.completed_at)::timestamptz AS last_completed
FROM production_order_operations poo
JOIN route_operations ro ON ro.id = poo.route_operation_id AND ro.route_id = sqlc.arg(route_id)
JOIN manufacturing_routes mr ON mr.id = ro.route_id AND mr.enterprise_id = sqlc.arg(enterprise_id)
WHERE poo.enterprise_id = sqlc.arg(enterprise_id)
  AND poo.status = 'DONE' AND poo.actual_hours > 0
  AND poo.completed_at >= NOW() - make_interval(days => sqlc.arg(days)::int)
GROUP BY poo.route_operation_id;
