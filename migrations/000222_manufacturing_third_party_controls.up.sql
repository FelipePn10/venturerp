ALTER TABLE items
    ADD COLUMN IF NOT EXISTS production_reporting_type VARCHAR(20) NOT NULL DEFAULT 'OPERATION',
    ADD COLUMN IF NOT EXISTS material_issue_timing VARCHAR(20) NOT NULL DEFAULT 'PRODUCTION';

ALTER TABLE operations
    ADD COLUMN IF NOT EXISTS third_party_remittance VARCHAR(20) NOT NULL DEFAULT 'DEMAND_ITEMS';
ALTER TABLE route_operations
    ADD COLUMN IF NOT EXISTS third_party_remittance VARCHAR(20);

ALTER TABLE items ADD CONSTRAINT chk_items_production_reporting_type
    CHECK (production_reporting_type IN ('ORDER','OPERATION'));
ALTER TABLE items ADD CONSTRAINT chk_items_material_issue_timing
    CHECK (material_issue_timing IN ('REGISTRATION_RELEASE','PRODUCTION'));
ALTER TABLE operations ADD CONSTRAINT chk_operations_third_party_remittance
    CHECK (third_party_remittance IN ('DEMAND_ITEMS','ORDER_ITEM','NONE'));
ALTER TABLE route_operations ADD CONSTRAINT chk_route_operations_third_party_remittance
    CHECK (third_party_remittance IS NULL OR third_party_remittance IN ('DEMAND_ITEMS','ORDER_ITEM','NONE'));
