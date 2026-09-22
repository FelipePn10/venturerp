BEGIN;
CREATE TABLE production_operation_execution_events (
 id BIGSERIAL PRIMARY KEY,
 enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
 production_order_id BIGINT NOT NULL,
 operation_id BIGINT NOT NULL,
 old_status TEXT NOT NULL,
 new_status TEXT NOT NULL,
 actual_hours_delta NUMERIC NOT NULL,
 actor_id UUID,
 source TEXT NOT NULL,
 notes TEXT,
 occurred_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX production_execution_events_order ON production_operation_execution_events(enterprise_id,production_order_id,operation_id,id);
CREATE FUNCTION record_production_execution_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.status IS DISTINCT FROM OLD.status OR NEW.actual_hours IS DISTINCT FROM OLD.actual_hours THEN
 INSERT INTO production_operation_execution_events(enterprise_id,production_order_id,operation_id,old_status,new_status,actual_hours_delta,actor_id,source,notes)
 SELECT ord.enterprise_id,NEW.production_order_id,NEW.id,OLD.status,NEW.status,NEW.actual_hours-OLD.actual_hours,
 NULLIF(current_setting('venture.execution_actor',true),'')::uuid,COALESCE(NULLIF(current_setting('venture.execution_source',true),''),'LEGACY'),NEW.notes
 FROM production_orders ord WHERE ord.id=NEW.production_order_id;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER production_execution_event AFTER UPDATE ON production_order_operations FOR EACH ROW EXECUTE FUNCTION record_production_execution_event();
CREATE FUNCTION protect_production_execution_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'O histórico de execução é imutável'; END $$;
CREATE TRIGGER protect_production_execution_event BEFORE UPDATE OR DELETE ON production_operation_execution_events FOR EACH ROW EXECUTE FUNCTION protect_production_execution_event();
COMMIT;
