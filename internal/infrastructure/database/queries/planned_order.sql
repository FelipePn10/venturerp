-- name: CreatePlannedOrder :one
INSERT INTO planned_orders (
    order_number,
    item_code,
    mask,
    quantity,
    quantity_loss,
    quantity_corrected,
    order_type,
    status,
    plan_code,
    demand_type,
    demand_code,
    need_date,
    start_date,
    end_date,
    cost_center_code,
    employee_code,
    machine_code,
    warehouse_code,
    inter_factory,
    source_enterprise_code,
    auto_release,
    mrp_suggestion_code,
    production_time,
    priority,
    llc,
    notes,
    parent_order_code,
    sales_order_code,
    created_by,
    enterprise_id
)
VALUES (
           $1, $2, $3, $4, $5, $6,
           $7, $8, $9, $10, $11, $12,
           $13, $14, $15, $16, $17, $18,
           $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, @enterprise_id
       )
    RETURNING *;

-- name: GetPlannedOrderByCode :one
SELECT * FROM planned_orders WHERE code = $1 AND enterprise_id = @enterprise_id;

-- name: GetPlannedOrderByNumber :one
SELECT * FROM planned_orders WHERE order_number = $1 AND enterprise_id = @enterprise_id;

-- name: GetPlannedOrderByMRPSuggestionCode :one
SELECT * FROM planned_orders WHERE mrp_suggestion_code = $1 AND enterprise_id = @enterprise_id;

-- name: ListPlannedOrders :many
SELECT * FROM planned_orders WHERE enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY need_date;

-- name: ListPlannedOrdersByPlan :many
SELECT * FROM planned_orders WHERE plan_code = $1 AND enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY need_date;

-- name: ListPlannedOrdersByItem :many
SELECT * FROM planned_orders WHERE item_code = $1 AND enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY need_date;

-- name: ListPlannedOrdersByType :many
SELECT * FROM planned_orders WHERE order_type = $1 AND enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY need_date;

-- name: ListPlannedOrdersByStatus :many
SELECT * FROM planned_orders WHERE status = $1 AND enterprise_id = @enterprise_id AND is_active = TRUE ORDER BY need_date;

-- name: UpdatePlannedOrderStatus :one
UPDATE planned_orders SET status = $1, updated_at = NOW() WHERE code = $2 AND enterprise_id = @enterprise_id RETURNING *;

-- name: FirmPlannedOrder :one
UPDATE planned_orders SET is_firm = TRUE, status = 'RELEASED', updated_at = NOW() WHERE code = $1 AND enterprise_id = @enterprise_id RETURNING *;

-- name: SetPlannedOrderPlanningState :one
UPDATE planned_orders
SET status = $1, is_firm = $2, updated_at = NOW()
WHERE code = $3 AND enterprise_id = @enterprise_id
RETURNING *;

-- name: IsPlannedOrderItemKanban :one
SELECT EXISTS (
    SELECT 1 FROM kanban_cards
    WHERE item_code = $1 AND enterprise_id = @enterprise_id AND status = 'ACTIVE'
) AS is_kanban;

-- name: HasPlannedOrderProductionMovements :one
SELECT EXISTS (
    SELECT 1
    FROM production_orders po
    JOIN planned_orders planned ON planned.id = po.planned_order_id
    WHERE planned.code = $1
      AND planned.enterprise_id = @enterprise_id
      AND (
          EXISTS (SELECT 1 FROM production_appointments pa WHERE pa.production_order_id = po.id)
          OR EXISTS (SELECT 1 FROM production_consumptions pc WHERE pc.production_order_id = po.id)
          OR EXISTS (
              SELECT 1 FROM stock_movements sm
              WHERE sm.reference_type = 'PRODUCTION_ORDER' AND sm.reference_code = po.id
          )
      )
) AS has_movements;

-- name: UpdatePlannedOrderDates :one
UPDATE planned_orders SET start_date = $1, end_date = $2, updated_at = NOW() WHERE code = $3 AND enterprise_id = @enterprise_id RETURNING *;

-- name: DeletePlannedOrder :exec
UPDATE planned_orders SET is_active = FALSE, updated_at = NOW() WHERE code = $1 AND enterprise_id = @enterprise_id;

-- name: GetNextOrderNumber :one
SELECT COALESCE(MAX(order_number), 0) + 1 AS next_number FROM planned_orders WHERE enterprise_id = @enterprise_id;

-- name: DeleteOrdersByPlan :exec
UPDATE planned_orders SET is_active = FALSE, updated_at = NOW() WHERE plan_code = $1 AND enterprise_id = @enterprise_id;

-- name: ListFirmPlannedOrdersByItems :many
-- O suprimento da ordem firme é o que ainda falta chegar. A OF encerrada já virou
-- estoque (ou, no beneficiamento, material a devolver) e a cancelada não vem mais:
-- contá-las cobria para sempre os pedidos seguintes do item, que o MRP não planejava.
SELECT po.code, po.item_code,
       GREATEST(po.quantity - COALESCE(prod.entregue, 0) - COALESCE(compra.chegou, 0), 0)::numeric AS quantity,
       po.need_date
FROM planned_orders po
LEFT JOIN LATERAL (
    SELECT SUM(o.produced_qty) AS entregue,
           BOOL_AND(o.status IN ('COMPLETED', 'CLOSED', 'CANCELLED')) AS encerrada
      FROM production_orders o
     WHERE o.planned_order_id = po.id AND o.enterprise_id = po.enterprise_id
) prod ON TRUE
-- O mesmo para a compra: o recebido já é estoque e o cancelado não vem mais.
LEFT JOIN LATERAL (
    SELECT SUM(poi.received_qty + poi.cancelled_qty) AS chegou
      FROM purchase_order_items poi
      JOIN purchase_orders pc ON pc.code = poi.purchase_order_code
      JOIN enterprise emp ON emp.code = pc.enterprise_code
     WHERE poi.planned_order_code = po.code AND emp.id = po.enterprise_id
) compra ON TRUE
WHERE po.item_code = ANY(@item_codes::bigint[])
  AND po.enterprise_id = @enterprise_id
  AND po.is_firm = TRUE
  AND po.is_active = TRUE
  -- A ordem de SERVIÇO (etapa em terceiro) não traz o item: o item volta pela
  -- OF da qual a etapa faz parte. Contá-la dobrava o suprimento — firmar a
  -- sugestão de fabricação e a de serviço da mesma peça dava 2× a peça.
  AND po.order_type <> 'OUTSOURCING'
  AND NOT COALESCE(prod.encerrada, FALSE)
  AND po.quantity > COALESCE(prod.entregue, 0) + COALESCE(compra.chegou, 0)
ORDER BY po.item_code, po.need_date;

-- name: ReschedulePlannedOrder :one
-- Move a necessidade da ordem e desloca início e fim pelo mesmo intervalo, para
-- preservar o prazo de fabricação/compra.
UPDATE planned_orders
   SET start_date = start_date + (sqlc.arg(need_date)::date - need_date),
       end_date   = end_date + (sqlc.arg(need_date)::date - need_date),
       need_date  = sqlc.arg(need_date)::date,
       updated_at = NOW()
 WHERE code = @code AND enterprise_id = @enterprise_id AND is_active
RETURNING *;

-- name: CancelPlannedOrder :one
UPDATE planned_orders
   SET status = 'CANCELLED', is_firm = FALSE, is_active = FALSE, updated_at = NOW()
 WHERE code = $1 AND enterprise_id = @enterprise_id AND is_active
RETURNING *;

-- name: GetPlannedOrderActiveProductionOrder :one
-- OF ainda viva gerada pela ordem planejada. O vínculo é pelo ID interno da
-- ordem planejada (production_orders.planned_order_id → planned_orders.id).
SELECT po.order_number
  FROM production_orders po
  JOIN planned_orders planned ON planned.id = po.planned_order_id
 WHERE planned.code = $1 AND planned.enterprise_id = @enterprise_id
   AND po.status NOT IN ('CANCELLED')
 ORDER BY po.id DESC
 LIMIT 1;

-- name: ListPurchasePlannedOrdersWithOrder :many
-- Ordens planejadas de compra que já viraram pedido (item de pedido não
-- cancelado apontando para elas). A fila do comprador (VPDC0200) é o resto:
-- firmada no MRP ou não, a compra só sai da fila quando vira pedido.
SELECT DISTINCT po.code
FROM planned_orders po
JOIN purchase_order_items poi ON poi.planned_order_code = po.code AND poi.status <> 'CANCELLED'
JOIN purchase_orders pc ON pc.code = poi.purchase_order_code AND pc.status <> 'CANCELLED'
JOIN enterprise emp ON emp.code = pc.enterprise_code AND emp.id = po.enterprise_id
WHERE po.enterprise_id = sqlc.arg(enterprise_id)
  AND po.order_type = 'PURCHASE';

-- name: ListOpenProductionOrderSupply :many
-- OF aberta que NÃO está coberta por ordem planejada firme (OF manual, da
-- demanda independente, de ordem planejada não firme): o que falta produzir é
-- suprimento do item na data de término. A OF de ordem firme já entra pela
-- ordem planejada (ListFirmPlannedOrdersByItems) — contar as duas dobrava.
SELECT o.id, o.item_code,
       GREATEST(o.planned_qty - o.produced_qty, 0)::numeric AS quantity,
       COALESCE(o.end_date, o.start_date, CURRENT_DATE)::date AS arrival_date
FROM production_orders o
WHERE o.enterprise_id = sqlc.arg(enterprise_id)
  AND o.item_code = ANY(sqlc.arg(item_codes)::bigint[])
  AND o.status NOT IN ('COMPLETED', 'CLOSED', 'CANCELLED')
  AND o.planned_qty > o.produced_qty
  AND NOT EXISTS (SELECT 1 FROM planned_orders p
                   WHERE p.id = o.planned_order_id AND p.enterprise_id = o.enterprise_id
                     AND p.is_firm AND p.is_active AND p.status <> 'CANCELLED');

-- name: ListOpenPurchaseOrderSupply :many
-- Item de pedido de compra em aberto que NÃO veio de ordem planejada firme
-- (pedido lançado pelo comprador): o saldo a receber, na unidade de estoque, é
-- suprimento na data prometida. Rascunho conta — o comprador já assumiu a
-- compra; não contar fazia o MRP sugerir de novo.
SELECT i.code, i.item_code,
       (GREATEST(i.requested_qty - i.received_qty - i.cancelled_qty, 0)
        * CASE WHEN COALESCE(i.internal_qty, 0) > 0 AND i.requested_qty > 0
               THEN i.internal_qty / i.requested_qty ELSE 1 END)::numeric AS quantity,
       COALESCE(i.promised_date, i.delivery_date, pc.delivery_date, CURRENT_DATE)::date AS arrival_date
FROM purchase_order_items i
JOIN purchase_orders pc ON pc.code = i.purchase_order_code
JOIN enterprise emp ON emp.code = pc.enterprise_code
WHERE emp.id = sqlc.arg(enterprise_id)
  AND i.item_code = ANY(sqlc.arg(item_codes)::bigint[])
  AND i.is_active
  AND i.status NOT IN ('CANCELLED', 'RECEIVED')
  AND pc.status NOT IN ('CANCELLED', 'RECEIVED')
  AND i.requested_qty > i.received_qty + i.cancelled_qty
  AND NOT EXISTS (SELECT 1 FROM planned_orders p
                   WHERE p.code = i.planned_order_code AND p.enterprise_id = emp.id
                     AND p.is_firm AND p.is_active AND p.status <> 'CANCELLED');

-- name: ListOpenProductionOrderAllocations :many
-- Empenho das OFs abertas: o material da lista que ainda não saiu do estoque
-- (necessário − atendido) é demanda dependente do componente. Sem isto, firmar
-- a ordem fazia a necessidade do componente sumir no MRP seguinte.
SELECT o.id AS production_order_id, o.item_code AS parent_item_code, o.mask AS parent_mask, o.planned_qty AS parent_quantity,
       COALESCE(m.substituted_item_code, m.item_code)::bigint AS item_code, m.mask,
       (m.quantity - m.attended_quantity)::numeric AS quantity,
       COALESCE(o.start_date, CURRENT_DATE)::date AS start_date,
       COALESCE(o.end_date, o.start_date, CURRENT_DATE)::date AS end_date,
       m.route_operation_sequence
FROM production_order_materials m
JOIN production_orders o ON o.id = m.production_order_id AND o.enterprise_id = m.enterprise_id
WHERE m.enterprise_id = sqlc.arg(enterprise_id)
  AND m.material_kind = 'DEMAND'
  AND o.status NOT IN ('COMPLETED', 'CLOSED', 'CANCELLED')
  AND m.quantity > m.attended_quantity
ORDER BY o.id, m.id;

-- name: ListFirmPlannedProductionWithoutOrder :many
-- Ordem planejada de fabricação firme que ainda não virou OF: o MRP a conta
-- como suprimento do item, então os componentes dela têm de ser demanda — a
-- estrutura é explodida pelo que falta. Firmar já cria a OF; isto cobre a
-- planejada firmada sem OF (OF cancelada fica de fora: a planejada morreu junto).
SELECT po.code, po.item_code, COALESCE(po.mask, '')::text AS mask,
       COALESCE(NULLIF(po.quantity_corrected, 0), po.quantity)::numeric AS quantity,
       COALESCE(po.start_date, po.need_date)::date AS start_date,
       po.need_date
FROM planned_orders po
WHERE po.enterprise_id = sqlc.arg(enterprise_id)
  AND po.order_type = 'PRODUCTION'
  AND po.is_firm AND po.is_active AND po.status <> 'CANCELLED'
  AND NOT EXISTS (SELECT 1 FROM production_orders o WHERE o.planned_order_id = po.id AND o.enterprise_id = po.enterprise_id);

-- name: ListOpenPlannedOrderExceptions :many
-- Mensagens de exceção do MRP ainda não tratadas que apontam uma ordem
-- planejada: a fila do comprador mostra "MRP recomenda cancelar/adiar" na
-- própria sugestão — antes ele só via isso no VMRP0100.
SELECT e.source_code, e.message_type, COALESCE(e.description, '')::text AS description, e.suggested_date
FROM mrp_exception_messages e
WHERE e.enterprise_id = sqlc.arg(enterprise_id)
  AND e.source_type = 'PLANNED_ORDER'
  AND e.handling_action IS NULL
  AND e.source_code IS NOT NULL
ORDER BY e.source_code, e.created_at DESC;
