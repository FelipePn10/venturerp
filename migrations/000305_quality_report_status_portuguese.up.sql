BEGIN;
ALTER TABLE item_supplier_quality_reports DROP CONSTRAINT IF EXISTS item_supplier_quality_reports_status_check;
UPDATE item_supplier_quality_reports SET status=CASE status WHEN 'PENDING' THEN 'PENDENTE' WHEN 'APPROVED' THEN 'APROVADO' WHEN 'REJECTED' THEN 'REJEITADO' WHEN 'EXPIRED' THEN 'EXPIRADO' ELSE status END;
ALTER TABLE item_supplier_quality_reports ADD CONSTRAINT chk_item_supplier_quality_status_pt CHECK(status IN ('PENDENTE','APROVADO','REJEITADO','EXPIRADO'));
COMMIT;
