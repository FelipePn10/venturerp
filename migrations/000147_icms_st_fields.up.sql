-- ICMS Substituição Tributária (ST) fields on fiscal exits and their items.
-- The tax engine now computes ST when an item carries an MVA; these columns
-- persist the result for SPED, the Focus NF-e payload and tax assessment.

ALTER TABLE public.fiscal_exits
    ADD COLUMN IF NOT EXISTS base_icms_st  NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_icms_st NUMERIC(15,2) NOT NULL DEFAULT 0;

ALTER TABLE public.fiscal_exit_items
    ADD COLUMN IF NOT EXISTS base_icms_st  NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS aliq_icms_st  NUMERIC(7,4)  NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_icms_st NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS mva           NUMERIC(7,4)  NOT NULL DEFAULT 0;
