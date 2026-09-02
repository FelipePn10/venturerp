DROP INDEX IF EXISTS public.ux_fiscal_configs_enterprise;

ALTER TABLE public.fiscal_configs
    DROP COLUMN IF EXISTS email,
    DROP COLUMN IF EXISTS trade_name,
    DROP COLUMN IF EXISTS enterprise_id;
