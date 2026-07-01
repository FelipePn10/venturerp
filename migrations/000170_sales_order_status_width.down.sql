BEGIN;

-- Revert to the original width. Any 'CANCELLED' rows are collapsed to the
-- short code 'C' first so the narrowing does not fail.
UPDATE public.sales_orders SET status = 'C' WHERE status = 'CANCELLED';

ALTER TABLE public.sales_orders
    ALTER COLUMN status TYPE VARCHAR(5);

COMMIT;
