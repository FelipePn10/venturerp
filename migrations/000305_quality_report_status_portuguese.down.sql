BEGIN;
ALTER TABLE item_supplier_quality_reports DROP CONSTRAINT IF EXISTS chk_item_supplier_quality_status_pt;
UPDATE item_supplier_quality_reports SET status=CASE status WHEN 'PENDENTE' THEN 'PENDING' WHEN 'APROVADO' THEN 'APPROVED' WHEN 'REJEITADO' THEN 'REJECTED' WHEN 'EXPIRADO' THEN 'EXPIRED' ELSE status END;
ALTER TABLE item_supplier_quality_reports ADD CONSTRAINT item_supplier_quality_reports_status_check CHECK(status IN ('PENDING','APPROVED','REJECTED','EXPIRED'));
COMMIT;
