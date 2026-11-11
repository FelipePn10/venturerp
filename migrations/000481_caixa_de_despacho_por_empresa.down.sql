ALTER TABLE shipment_dispatch_boxes DROP CONSTRAINT IF EXISTS shipment_dispatch_boxes_enterprise_code_key;
CREATE INDEX IF NOT EXISTS ix_dispatch_boxes_tenant_code ON shipment_dispatch_boxes(enterprise_id, code);
-- Volta à unicidade global; falha se duas empresas já usarem o mesmo código.
ALTER TABLE shipment_dispatch_boxes ADD CONSTRAINT shipment_dispatch_boxes_code_key UNIQUE (code);
