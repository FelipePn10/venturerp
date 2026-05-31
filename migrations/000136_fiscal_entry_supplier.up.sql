BEGIN;

-- Link a purchase NF entry to the registered supplier (matched by emitter
-- CNPJ/CPF on import). FK-less BIGINT to match the fiscal module convention
-- (icms_reduction_substitutions, etc.).
ALTER TABLE public.fiscal_entries
    ADD COLUMN IF NOT EXISTS supplier_code BIGINT;

CREATE INDEX IF NOT EXISTS ix_fiscal_entries_supplier ON public.fiscal_entries(supplier_code)
    WHERE supplier_code IS NOT NULL;

COMMIT;
