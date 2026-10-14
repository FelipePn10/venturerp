ALTER TABLE mrp_exception_messages DROP CONSTRAINT IF EXISTS ck_mrp_exception_handling_action;
ALTER TABLE mrp_exception_messages
    DROP COLUMN IF EXISTS handling_note,
    DROP COLUMN IF EXISTS handling_action,
    DROP COLUMN IF EXISTS handled_by,
    DROP COLUMN IF EXISTS handled_at,
    DROP COLUMN IF EXISTS suggested_date;
