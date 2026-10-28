-- Volta à unicidade global (falha se duas empresas já tiverem o mesmo número).
ALTER TABLE shipment_dispatch_boxes DROP CONSTRAINT IF EXISTS shipment_dispatch_boxes_current_load_fkey;
ALTER TABLE shipment_loads DROP CONSTRAINT IF EXISTS shipment_loads_enterprise_code_key;
ALTER TABLE shipment_loads ADD CONSTRAINT shipment_loads_code_key UNIQUE (code);
ALTER TABLE shipments DROP CONSTRAINT IF EXISTS shipments_enterprise_code_key;
ALTER TABLE shipments ADD CONSTRAINT shipments_code_key UNIQUE (code);
ALTER TABLE shipment_dispatch_boxes ADD CONSTRAINT shipment_dispatch_boxes_current_load_fkey
    FOREIGN KEY (current_load) REFERENCES shipment_loads (code) ON DELETE SET NULL;
SELECT instalar_conferencia_de_empresa();
