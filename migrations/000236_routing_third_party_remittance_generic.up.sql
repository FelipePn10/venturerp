BEGIN;

ALTER TABLE operations DROP CONSTRAINT IF EXISTS chk_operations_third_party_remittance;
ALTER TABLE route_operations DROP CONSTRAINT IF EXISTS chk_route_operations_third_party_remittance;
ALTER TABLE operations ADD CONSTRAINT chk_operations_third_party_remittance
    CHECK (third_party_remittance IN ('DEMAND_ITEMS','ORDER_ITEM','GENERIC','NONE'));
ALTER TABLE route_operations ADD CONSTRAINT chk_route_operations_third_party_remittance
    CHECK (third_party_remittance IS NULL OR third_party_remittance IN ('DEMAND_ITEMS','ORDER_ITEM','GENERIC','NONE'));

COMMIT;
