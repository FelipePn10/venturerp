BEGIN;

-- The order-level cancel writes 'CANCELLED' (9 chars) but the column was
-- VARCHAR(5) (sized for the short lifecycle codes R/P/A/OA/OF/F), causing
-- "value too long for type character varying(5)" on DELETE /cancel.
-- Widen to VARCHAR(20) to match the item status column.
ALTER TABLE public.sales_orders
    ALTER COLUMN status TYPE VARCHAR(20);

COMMIT;
