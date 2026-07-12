ALTER TABLE route_operations DROP CONSTRAINT IF EXISTS chk_route_operations_third_party_remittance;
ALTER TABLE operations DROP CONSTRAINT IF EXISTS chk_operations_third_party_remittance;
ALTER TABLE items DROP CONSTRAINT IF EXISTS chk_items_material_issue_timing;
ALTER TABLE items DROP CONSTRAINT IF EXISTS chk_items_production_reporting_type;
ALTER TABLE route_operations DROP COLUMN IF EXISTS third_party_remittance;
ALTER TABLE operations DROP COLUMN IF EXISTS third_party_remittance;
ALTER TABLE items DROP COLUMN IF EXISTS material_issue_timing;
ALTER TABLE items DROP COLUMN IF EXISTS production_reporting_type;
