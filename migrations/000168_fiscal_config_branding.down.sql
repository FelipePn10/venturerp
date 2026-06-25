ALTER TABLE public.fiscal_configs
    DROP COLUMN IF EXISTS logo,
    DROP COLUMN IF EXISTS logo_mime,
    DROP COLUMN IF EXISTS brand_color;
