ALTER TABLE delivery_reschedules
    ADD COLUMN IF NOT EXISTS code BIGINT;

UPDATE delivery_reschedules
SET code = id
WHERE code IS NULL;

ALTER TABLE delivery_reschedules
    ALTER COLUMN code SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'delivery_reschedules_code_key'
    ) THEN
ALTER TABLE delivery_reschedules
    ADD CONSTRAINT delivery_reschedules_code_key UNIQUE (code);
END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_delivery_reschedules_code
    ON delivery_reschedules (code);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'delivery_reschedules'
          AND column_name = 'sales_order_id'
    ) THEN
ALTER TABLE delivery_reschedules
    RENAME COLUMN sales_order_id TO sales_order_code;
END IF;
END $$;

DROP INDEX IF EXISTS idx_delivery_reschedule_order;

CREATE INDEX IF NOT EXISTS idx_delivery_reschedule_order_code
    ON delivery_reschedules (sales_order_code);