CREATE TABLE production_order_service_requisition_links (
    production_order_id BIGINT NOT NULL REFERENCES production_orders(id) ON DELETE CASCADE,
    purchase_requisition_code BIGINT NOT NULL REFERENCES purchase_requisitions(code) ON DELETE CASCADE,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    PRIMARY KEY (enterprise_id, production_order_id, purchase_requisition_code)
);

CREATE INDEX idx_service_requisition_link_requisition
    ON production_order_service_requisition_links (enterprise_id, purchase_requisition_code);
