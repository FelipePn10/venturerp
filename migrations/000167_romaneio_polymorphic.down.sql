BEGIN;

ALTER TABLE shipments
  DROP COLUMN IF EXISTS reference_type,
  DROP COLUMN IF EXISTS purchase_order_code,
  DROP COLUMN IF EXISTS production_order_code;

COMMIT;
