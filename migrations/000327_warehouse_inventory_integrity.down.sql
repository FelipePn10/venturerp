BEGIN;
DO $$ DECLARE table_name TEXT; BEGIN
 FOREACH table_name IN ARRAY ARRAY['warehouse','physical_inventories','physical_inventory_items','stock_movements','stock_balances','lot_masks','shipments','shipment_loads','shipment_delivery_instructions','shipment_dispatch_boxes'] LOOP
  EXECUTE format('DROP TRIGGER IF EXISTS trg_%s_inventory_audit ON %I',table_name,table_name);
 END LOOP;
END $$;
DROP TRIGGER IF EXISTS trg_warehouse_inventory_audit_immutable ON warehouse_inventory_audit;
DROP FUNCTION IF EXISTS prevent_warehouse_inventory_audit_change();
DROP FUNCTION IF EXISTS audit_warehouse_inventory_change();
DROP TABLE IF EXISTS warehouse_inventory_audit;
DO $$ BEGIN
 IF to_regclass('stock_cycle_counts') IS NOT NULL THEN
  ALTER TABLE stock_cycle_counts DROP CONSTRAINT IF EXISTS fk_cycle_count_physical_address;
  ALTER TABLE stock_cycle_counts DROP CONSTRAINT IF EXISTS ck_cycle_count_address_positive;
 END IF;
END $$;
DROP INDEX IF EXISTS ix_dispatch_boxes_tenant_code;
DROP INDEX IF EXISTS ix_shipment_loads_tenant_code;
DROP INDEX IF EXISTS ix_shipments_tenant_code;
ALTER TABLE shipment_dispatch_boxes DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE shipment_delivery_instructions DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE shipment_loads DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE shipments DROP COLUMN IF EXISTS enterprise_id;
DROP INDEX IF EXISTS ix_lot_masks_tenant_context;
ALTER TABLE lot_masks DROP COLUMN IF EXISTS enterprise_id;
DROP INDEX IF EXISTS ix_warehouse_tenant_type;
ALTER TABLE warehouse DROP COLUMN IF EXISTS enterprise_id;
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM warehouse WHERE location::TEXT IN ('EXPEDICAO','ASSISTENCIA_TECNICA')) THEN
	 RAISE EXCEPTION 'rollback 000327 recusado: existem almoxarifados com localizações sem representação no schema anterior';
	END IF;
	 ALTER TABLE warehouse ALTER COLUMN location TYPE TEXT USING location::TEXT;
 ALTER TABLE warehouse ALTER COLUMN type TYPE TEXT USING type::TEXT;
 UPDATE warehouse SET location=type,type=location;
 DROP TYPE warehouse_location;
 DROP TYPE warehouse_type;
 CREATE TYPE warehouse_location AS ENUM ('LINHA_DE_PRODUCAO','NORMAL');
 CREATE TYPE warehouse_type AS ENUM ('INTERNO','EXTERNO','ASSISTENCIA','REJEICAO','INSPECAO','RESERVA','TRANSITO','ESPECIAL');
 ALTER TABLE warehouse ALTER COLUMN location TYPE warehouse_location USING location::warehouse_location;
 ALTER TABLE warehouse ALTER COLUMN type TYPE warehouse_type USING type::warehouse_type;
END
$$;
COMMIT;
