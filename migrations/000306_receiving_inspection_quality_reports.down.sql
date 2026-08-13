DROP TABLE IF EXISTS receiving_inspection_quality_reports;
DROP FUNCTION IF EXISTS validate_receiving_inspection_quality_report();
ALTER TABLE item_supplier_quality_reports
    DROP CONSTRAINT IF EXISTS uq_item_supplier_quality_reports_enterprise_id_id;
