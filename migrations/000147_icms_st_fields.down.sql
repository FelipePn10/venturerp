ALTER TABLE public.fiscal_exit_items
    DROP COLUMN IF EXISTS base_icms_st,
    DROP COLUMN IF EXISTS aliq_icms_st,
    DROP COLUMN IF EXISTS valor_icms_st,
    DROP COLUMN IF EXISTS mva;

ALTER TABLE public.fiscal_exits
    DROP COLUMN IF EXISTS base_icms_st,
    DROP COLUMN IF EXISTS valor_icms_st;
