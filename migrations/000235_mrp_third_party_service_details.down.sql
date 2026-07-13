BEGIN;

DROP INDEX IF EXISTS idx_mrp_service_suggestions_query;
ALTER TABLE mrp_planned_suggestions
    DROP CONSTRAINT IF EXISTS chk_mrp_service_suggestion_details,
    DROP COLUMN IF EXISTS remittance_type,
    DROP COLUMN IF EXISTS service_item_code,
    DROP COLUMN IF EXISTS supplier_code,
    DROP COLUMN IF EXISTS operation_id,
    DROP COLUMN IF EXISTS route_operation_id,
    DROP COLUMN IF EXISTS mask;

COMMIT;
