ALTER TABLE delivery_reschedules
    ADD COLUMN IF NOT EXISTS sales_order_code BIGINT;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'delivery_reschedules'
          AND column_name = 'sales_order_id'
    ) THEN

        EXECUTE '
            UPDATE delivery_reschedules
            SET sales_order_code = sales_order_id
            WHERE sales_order_code IS NULL
        ';

EXECUTE '
            ALTER TABLE delivery_reschedules
            DROP COLUMN sales_order_id
        ';
END IF;
END $$;