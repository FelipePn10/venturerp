BEGIN;
ALTER TABLE production_orders ADD COLUMN routing_snapshot_at TIMESTAMPTZ;
CREATE TABLE production_order_operation_dependencies (
 production_order_id BIGINT NOT NULL REFERENCES production_orders(id) ON DELETE CASCADE,
 enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
 predecessor_id BIGINT NOT NULL REFERENCES production_order_operations(id) ON DELETE CASCADE,
 successor_id BIGINT NOT NULL REFERENCES production_order_operations(id) ON DELETE CASCADE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY(predecessor_id,successor_id),
 CHECK(predecessor_id<>successor_id)
);
CREATE INDEX production_operation_dependencies_order ON production_order_operation_dependencies(enterprise_id,production_order_id);
CREATE INDEX production_operation_dependencies_successor ON production_order_operation_dependencies(successor_id);
-- Preserve the currently applicable graph of existing orders at migration time.
INSERT INTO production_order_operation_dependencies(production_order_id,enterprise_id,predecessor_id,successor_id)
SELECT ord.id,ord.enterprise_id,prior.id,current.id
FROM production_orders ord
JOIN production_order_operations current ON current.production_order_id=ord.id
JOIN production_order_operations prior ON prior.production_order_id=ord.id
LEFT JOIN route_operations step ON step.id=current.route_operation_id
WHERE (
 EXISTS(SELECT 1 FROM route_operation_network edge WHERE edge.predecessor_id=prior.route_operation_id AND edge.successor_id=current.route_operation_id)
 OR (NOT EXISTS(SELECT 1 FROM route_operation_network edge JOIN route_operations source ON source.id=edge.predecessor_id WHERE source.route_id=step.route_id) AND prior.sequence<current.sequence)
);
UPDATE production_orders ord SET routing_snapshot_at=NOW() WHERE EXISTS(SELECT 1 FROM production_order_operations op WHERE op.production_order_id=ord.id);
COMMIT;
