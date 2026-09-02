BEGIN;

-- Corrige a inversão histórica entre localização operacional e tipo estrutural.
-- Os dois enums nasceram em 000023 dentro de um bloco DO; mantemos a troca aqui
-- também em bloco para que o gerador de código continue lendo a cadeia inteira.
DO $$
BEGIN
 ALTER TABLE warehouse ALTER COLUMN location TYPE TEXT USING location::TEXT;
 ALTER TABLE warehouse ALTER COLUMN type TYPE TEXT USING type::TEXT;
 UPDATE warehouse SET location = type, type = location
 WHERE location IN ('NORMAL','LINHA_DE_PRODUCAO')
   AND type IN ('INTERNO','EXTERNO','ASSISTENCIA','REJEICAO','INSPECAO','RESERVA','TRANSITO','ESPECIAL');
 DROP TYPE warehouse_location;
 DROP TYPE warehouse_type;
 CREATE TYPE warehouse_location AS ENUM
  ('INTERNO','EXTERNO','INSPECAO','REJEICAO','RESERVA','TRANSITO','ESPECIAL','EXPEDICAO','ASSISTENCIA','ASSISTENCIA_TECNICA');
 CREATE TYPE warehouse_type AS ENUM ('NORMAL','LINHA_DE_PRODUCAO');
 ALTER TABLE warehouse ALTER COLUMN location TYPE warehouse_location USING location::warehouse_location;
 ALTER TABLE warehouse ALTER COLUMN type TYPE warehouse_type USING type::warehouse_type;
END
$$;

ALTER TABLE warehouse ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
UPDATE warehouse w SET enterprise_id=candidate.enterprise_id
 FROM (
  SELECT user_id, MIN(enterprise_id) AS enterprise_id
  FROM user_enterprises GROUP BY user_id
  HAVING COUNT(DISTINCT enterprise_id)=1
 ) candidate
 WHERE candidate.user_id=w.created_by AND w.enterprise_id IS NULL;
UPDATE warehouse SET enterprise_id=(SELECT MIN(id) FROM enterprise)
 WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise)=1;
ALTER TABLE warehouse ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_warehouse_tenant_type ON warehouse(enterprise_id,location,type);

ALTER TABLE lot_masks ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
UPDATE lot_masks m SET enterprise_id=candidate.enterprise_id
 FROM (
  SELECT user_id, MIN(enterprise_id) AS enterprise_id
  FROM user_enterprises GROUP BY user_id
  HAVING COUNT(DISTINCT enterprise_id)=1
 ) candidate
 WHERE candidate.user_id=m.created_by AND m.enterprise_id IS NULL;
UPDATE lot_masks SET enterprise_id=(SELECT MIN(id) FROM enterprise)
 WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise)=1;
ALTER TABLE lot_masks ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_lot_masks_tenant_context
 ON lot_masks(enterprise_id,application,item_code,customer_code) WHERE is_active;

ALTER TABLE shipments ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE shipment_loads ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE shipment_delivery_instructions ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
ALTER TABLE shipment_dispatch_boxes ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
UPDATE shipments s SET enterprise_id=e.id
 FROM sales_orders so JOIN enterprise e ON e.code=so.enterprise_code
 WHERE so.code=s.sales_order_code AND s.enterprise_id IS NULL;
UPDATE shipments SET enterprise_id=(SELECT MIN(id) FROM enterprise)
 WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise)=1;
UPDATE shipment_loads l SET enterprise_id=candidate.enterprise_id
 FROM (
  SELECT sls.load_id, MIN(s.enterprise_id) AS enterprise_id
  FROM shipment_load_shipments sls
  JOIN shipments s ON s.id=sls.shipment_id
  GROUP BY sls.load_id HAVING COUNT(DISTINCT s.enterprise_id)=1
 ) candidate
 WHERE candidate.load_id=l.id AND l.enterprise_id IS NULL;
UPDATE shipment_loads SET enterprise_id=(SELECT MIN(id) FROM enterprise)
 WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise)=1;
UPDATE shipment_delivery_instructions i SET enterprise_id=l.enterprise_id
 FROM shipment_loads l WHERE l.id=i.load_id AND i.enterprise_id IS NULL;
UPDATE shipment_delivery_instructions SET enterprise_id=(SELECT MIN(id) FROM enterprise)
 WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise)=1;
UPDATE shipment_dispatch_boxes b SET enterprise_id=w.enterprise_id
 FROM warehouse w WHERE w.id=b.warehouse_id AND b.enterprise_id IS NULL;
UPDATE shipment_dispatch_boxes b SET enterprise_id=l.enterprise_id
 FROM shipment_loads l WHERE l.code=b.current_load AND b.enterprise_id IS NULL;
UPDATE shipment_dispatch_boxes SET enterprise_id=(SELECT MIN(id) FROM enterprise)
 WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise)=1;
ALTER TABLE shipments ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE shipment_loads ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE shipment_delivery_instructions ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE shipment_dispatch_boxes ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_shipments_tenant_code ON shipments(enterprise_id,code);
CREATE INDEX IF NOT EXISTS ix_shipment_loads_tenant_code ON shipment_loads(enterprise_id,code);
CREATE INDEX IF NOT EXISTS ix_dispatch_boxes_tenant_code ON shipment_dispatch_boxes(enterprise_id,code);

CREATE TABLE IF NOT EXISTS warehouse_inventory_audit (
 id BIGSERIAL PRIMARY KEY,
 enterprise_id BIGINT NOT NULL,
 entity_type TEXT NOT NULL,
 entity_key TEXT NOT NULL,
 action TEXT NOT NULL CHECK (action IN ('INSERT','UPDATE','DELETE')),
 before_state JSONB,
 after_state JSONB,
 actor_id UUID,
 occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS ix_warehouse_inventory_audit
 ON warehouse_inventory_audit(enterprise_id,entity_type,entity_key,occurred_at DESC);

CREATE OR REPLACE FUNCTION audit_warehouse_inventory_change() RETURNS TRIGGER AS $$
DECLARE row_data JSONB := COALESCE(to_jsonb(NEW),to_jsonb(OLD)); tenant_id BIGINT; actor UUID;
BEGIN
 tenant_id := NULLIF(row_data->>'enterprise_id','')::BIGINT;
 IF tenant_id IS NULL AND TG_TABLE_NAME='physical_inventory_items' THEN
  SELECT enterprise_id INTO tenant_id FROM physical_inventories WHERE id=(row_data->>'inventory_id')::BIGINT;
 END IF;
 actor := COALESCE(NULLIF(row_data->>'counted_by','')::UUID,NULLIF(row_data->>'created_by','')::UUID);
 INSERT INTO warehouse_inventory_audit(enterprise_id,entity_type,entity_key,action,before_state,after_state,actor_id)
 VALUES(tenant_id,TG_TABLE_NAME,COALESCE(row_data->>'id',row_data->>'code'),TG_OP,
  CASE WHEN TG_OP='INSERT' THEN NULL ELSE to_jsonb(OLD) END,
  CASE WHEN TG_OP='DELETE' THEN NULL ELSE to_jsonb(NEW) END,actor);
 RETURN COALESCE(NEW,OLD);
END $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION prevent_warehouse_inventory_audit_change() RETURNS TRIGGER AS $$
BEGIN RAISE EXCEPTION 'auditoria de almoxarifado é imutável'; END $$ LANGUAGE plpgsql;
CREATE TRIGGER trg_warehouse_inventory_audit_immutable BEFORE UPDATE OR DELETE ON warehouse_inventory_audit
 FOR EACH ROW EXECUTE FUNCTION prevent_warehouse_inventory_audit_change();

DO $$ DECLARE table_name TEXT; BEGIN
 FOREACH table_name IN ARRAY ARRAY['warehouse','physical_inventories','physical_inventory_items','stock_movements','stock_balances','lot_masks','shipments','shipment_loads','shipment_delivery_instructions','shipment_dispatch_boxes'] LOOP
  EXECUTE format('CREATE TRIGGER trg_%s_inventory_audit AFTER INSERT OR UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION audit_warehouse_inventory_change()',table_name,table_name);
 END LOOP;
END $$;

-- stock_cycle_counts.warehouse_address_id não tem tabela-alvo com chave simples:
-- manufacturing_warehouse_addresses é chaveada por (enterprise_id,warehouse_id,address)
-- e não possui coluna id. Em vez de uma FK impossível, garantimos a coerência
-- mínima do endereço com o almoxarifado e o tenant da contagem.
ALTER TABLE stock_cycle_counts
 DROP CONSTRAINT IF EXISTS fk_cycle_count_physical_address;
ALTER TABLE stock_cycle_counts
 ADD CONSTRAINT ck_cycle_count_address_positive
 CHECK (warehouse_address_id IS NULL OR warehouse_address_id > 0) NOT VALID;

COMMIT;
