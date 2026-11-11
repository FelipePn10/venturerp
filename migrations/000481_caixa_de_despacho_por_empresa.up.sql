-- O código da caixa de despacho é por empresa: o cadastro grava com
-- ON CONFLICT (enterprise_id, code), mas a tabela só tinha UNIQUE (code) global
-- (migração 196) — todo cadastro de caixa caía em erro 42P10 ("erro interno"),
-- e a segunda empresa não podia ter a sua DOCA-1.
ALTER TABLE shipment_dispatch_boxes DROP CONSTRAINT IF EXISTS shipment_dispatch_boxes_code_key;
DROP INDEX IF EXISTS ix_dispatch_boxes_tenant_code;
ALTER TABLE shipment_dispatch_boxes
    ADD CONSTRAINT shipment_dispatch_boxes_enterprise_code_key UNIQUE (enterprise_id, code);
