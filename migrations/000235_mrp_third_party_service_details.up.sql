BEGIN;

ALTER TABLE mrp_planned_suggestions
    ADD COLUMN mask VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN route_operation_id BIGINT REFERENCES route_operations(id),
    ADD COLUMN operation_id BIGINT REFERENCES operations(id),
    ADD COLUMN supplier_code BIGINT,
    ADD COLUMN service_item_code BIGINT,
    ADD COLUMN remittance_type VARCHAR(20);

ALTER TABLE mrp_planned_suggestions
    ADD CONSTRAINT chk_mrp_service_suggestion_details CHECK (
        (order_type <> 'SERVICO') OR
        (route_operation_id IS NOT NULL AND operation_id IS NOT NULL AND
         remittance_type IN ('DEMAND_ITEMS','ORDER_ITEM','GENERIC','NONE'))
    ) NOT VALID;

CREATE INDEX idx_mrp_service_suggestions_query
    ON mrp_planned_suggestions
       (enterprise_id, plan_code, operation_id, supplier_code, need_date)
    WHERE order_type = 'SERVICO';

COMMIT;
