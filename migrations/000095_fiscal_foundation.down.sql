BEGIN;

DROP TABLE IF EXISTS public.fiscal_exit_items;
DROP TABLE IF EXISTS public.fiscal_entry_items;
DROP TABLE IF EXISTS public.fiscal_exits;
DROP TABLE IF EXISTS public.fiscal_entries;
DROP TABLE IF EXISTS public.fiscal_configs;
DROP TABLE IF EXISTS public.icms_interstate;
DROP TABLE IF EXISTS public.icms_internal;
DROP TABLE IF EXISTS public.tax_scenarios;
DROP TABLE IF EXISTS public.ncm_tax_table;

COMMIT;
