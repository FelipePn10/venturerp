BEGIN;

DELETE FROM public.ncm_tax_table WHERE ncm IN (
    -- Placeholder: delete the NCMs that were inserted by the up migration.
    -- '0000.00.00'
);

COMMIT;
