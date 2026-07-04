BEGIN;

ALTER TABLE item_structures
    DROP COLUMN IF EXISTS is_coproduct,
    DROP COLUMN IF EXISTS is_fixed_qty;

COMMIT;
