BEGIN;

INSERT INTO public.ncm_tax_table (ncm, aliq_ipi)
VALUES
    -- Missing NCMs from the original 96-item spec (2 NCMs were not seeded in migration 95).
    -- TODO: identify the exact missing NCMs from the original spec and uncomment/fill below.
    -- ('0000.00.00', 0)
ON CONFLICT (ncm) DO NOTHING;

COMMIT;
