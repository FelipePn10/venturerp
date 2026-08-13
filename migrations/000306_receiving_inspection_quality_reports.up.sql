BEGIN;

ALTER TABLE item_supplier_quality_reports
    ADD CONSTRAINT uq_item_supplier_quality_reports_enterprise_id_id UNIQUE (enterprise_id, id);

CREATE TABLE receiving_inspection_quality_reports (
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    inspection_order_id BIGINT NOT NULL REFERENCES receiving_inspection_orders(id) ON DELETE CASCADE,
    quality_report_id BIGINT NOT NULL,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    linked_by UUID NOT NULL,
    PRIMARY KEY (inspection_order_id, quality_report_id),
    FOREIGN KEY (enterprise_id, quality_report_id)
        REFERENCES item_supplier_quality_reports(enterprise_id, id) ON DELETE RESTRICT
);

CREATE FUNCTION validate_receiving_inspection_quality_report() RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM receiving_inspection_orders o
        JOIN items i ON i.code=o.item_code AND i.enterprise_id=NEW.enterprise_id
        JOIN item_supplier_quality_reports q
          ON q.id=NEW.quality_report_id AND q.enterprise_id=NEW.enterprise_id
        JOIN item_preferred_suppliers s
          ON s.id=q.item_supplier_id
         AND s.enterprise_id=NEW.enterprise_id
         AND s.item_code=o.item_code
         AND (o.supplier_code IS NULL OR s.supplier_code=o.supplier_code)
        WHERE o.id=NEW.inspection_order_id
    ) THEN
        RAISE EXCEPTION 'laudo incompatível com a empresa, o item ou o fornecedor da inspeção'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_validate_receiving_inspection_quality_report
BEFORE INSERT OR UPDATE ON receiving_inspection_quality_reports
FOR EACH ROW EXECUTE FUNCTION validate_receiving_inspection_quality_report();

CREATE INDEX ix_receiving_inspection_quality_reports_tenant
    ON receiving_inspection_quality_reports(enterprise_id, inspection_order_id, linked_at DESC);

CREATE INDEX ix_receiving_inspection_quality_reports_report
    ON receiving_inspection_quality_reports(enterprise_id, quality_report_id);

COMMIT;
