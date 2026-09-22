-- name: LockOperationExecution :one
SELECT op.id, op.production_order_id, op.status, ord.status AS order_status
FROM production_order_operations op
JOIN production_orders ord ON ord.id = op.production_order_id
WHERE op.id = $1 AND ord.enterprise_id = $2
FOR UPDATE OF ord, op;

-- name: OperationPredecessorsPending :one
SELECT CASE WHEN (SELECT ord.routing_snapshot_at IS NOT NULL FROM production_orders ord JOIN production_order_operations op ON op.production_order_id=ord.id WHERE op.id=$1) THEN EXISTS(SELECT 1 FROM production_order_operation_dependencies d JOIN production_order_operations prior ON prior.id=d.predecessor_id WHERE d.successor_id=$1 AND prior.status NOT IN ('DONE','SKIPPED')) ELSE EXISTS (
 SELECT 1 FROM production_order_operations prior
 JOIN production_order_operations current ON current.production_order_id = prior.production_order_id
 LEFT JOIN route_operations step ON step.id = current.route_operation_id
 WHERE current.id = $1 AND prior.status NOT IN ('DONE', 'SKIPPED')
 AND (
   (EXISTS (SELECT 1 FROM route_operation_network edge WHERE edge.predecessor_id = prior.route_operation_id AND edge.successor_id = current.route_operation_id))
   OR (NOT EXISTS (SELECT 1 FROM route_operation_network edge JOIN route_operations source ON source.id = edge.predecessor_id WHERE source.route_id = step.route_id) AND prior.sequence < current.sequence)
 )
) END::boolean AS blocked;

-- name: UpdateOperationExecution :exec
UPDATE production_order_operations SET status = sqlc.arg(status)::text,
 started_at = CASE WHEN sqlc.arg(status)::text = 'IN_PROGRESS' THEN COALESCE(started_at, NOW()) ELSE started_at END,
 completed_at = CASE WHEN sqlc.arg(status)::text IN ('DONE','SKIPPED') THEN NOW() ELSE completed_at END,
 actual_hours = actual_hours + sqlc.arg(hours)::numeric,
 notes = concat_ws(E'\n', notes, to_char(NOW(), 'YYYY-MM-DD HH24:MI:SS TZ') || ' · ' || CASE sqlc.arg(status)::text WHEN 'IN_PROGRESS' THEN 'Em andamento' WHEN 'PAUSED' THEN 'Pausada' WHEN 'INTERRUPTED' THEN 'Interrompida' WHEN 'DONE' THEN 'Concluída' ELSE 'Dispensada' END || CASE WHEN sqlc.arg(reason)::text <> '' THEN ': ' || sqlc.arg(reason)::text ELSE '' END),
 updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: ProductionOperationsPending :one
SELECT EXISTS(SELECT 1 FROM production_order_operations WHERE production_order_id = $1 AND status NOT IN ('DONE','SKIPPED'))::boolean;

-- name: LockProductionExecution :one
SELECT id, status, item_code, COALESCE(mask,'')::text AS mask FROM production_orders WHERE id = $1 AND enterprise_id = $2 FOR UPDATE;

-- name: GetProductionExecution :one
SELECT id, status FROM production_orders WHERE id = $1 AND enterprise_id = $2;

-- name: OperationHasSuccessors :one
SELECT CASE WHEN (SELECT ord.routing_snapshot_at IS NOT NULL FROM production_orders ord JOIN production_order_operations op ON op.production_order_id=ord.id WHERE op.id=$1) THEN EXISTS(SELECT 1 FROM production_order_operation_dependencies WHERE predecessor_id=$1) ELSE EXISTS (
 SELECT 1 FROM production_order_operations next
 JOIN production_order_operations current ON current.production_order_id = next.production_order_id
 LEFT JOIN route_operations step ON step.id = current.route_operation_id
 WHERE current.id = $1 AND (
  EXISTS (SELECT 1 FROM route_operation_network edge WHERE edge.predecessor_id = current.route_operation_id AND edge.successor_id = next.route_operation_id)
  OR (NOT EXISTS (SELECT 1 FROM route_operation_network edge JOIN route_operations source ON source.id=edge.predecessor_id WHERE source.route_id=step.route_id) AND next.sequence > current.sequence)
 )
) END::boolean;

-- name: FreezeProductionRoutingDependencies :exec
INSERT INTO production_order_operation_dependencies(production_order_id,enterprise_id,predecessor_id,successor_id)
SELECT ord.id,ord.enterprise_id,prior.id,current.id
FROM production_orders ord
JOIN production_order_operations current ON current.production_order_id=ord.id
JOIN production_order_operations prior ON prior.production_order_id=ord.id
LEFT JOIN route_operations step ON step.id=current.route_operation_id
WHERE (
 EXISTS(SELECT 1 FROM route_operation_network edge WHERE edge.predecessor_id=prior.route_operation_id AND edge.successor_id=current.route_operation_id)
 OR (NOT EXISTS(SELECT 1 FROM route_operation_network edge JOIN route_operations source ON source.id=edge.predecessor_id WHERE source.route_id=step.route_id) AND prior.sequence<current.sequence)
) AND ord.id=$1 AND ord.enterprise_id=$2 AND ord.routing_snapshot_at IS NULL
ON CONFLICT DO NOTHING;

-- name: MarkProductionRoutingFrozen :exec
UPDATE production_orders SET routing_snapshot_at=COALESCE(routing_snapshot_at,NOW()) WHERE id=$1 AND enterprise_id=$2;

-- name: SetProductionExecutionActor :exec
SELECT set_config('venture.execution_actor',sqlc.arg(actor)::text,true),set_config('venture.execution_source',sqlc.arg(source)::text,true);

-- name: ListProductionExecutionEvents :many
SELECT e.id,e.old_status,e.new_status,e.actual_hours_delta,e.notes,e.occurred_at,
 COALESCE(u.name, e.actor_id::text, 'Sistema (legado)')::text AS actor_name
FROM production_operation_execution_events e
JOIN production_orders ord ON ord.id=e.production_order_id AND ord.enterprise_id=e.enterprise_id
LEFT JOIN users u ON u.id=e.actor_id
WHERE e.operation_id=$1 AND e.enterprise_id=$2
ORDER BY e.id;
