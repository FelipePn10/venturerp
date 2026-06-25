-- Company branding for professional report letterheads (logo + brand colour).
-- The logo is stored inline as bytes so report generation needs no filesystem or
-- object store; logos are small. brand_color is a #RRGGBB hex used to tint the
-- PDF letterhead band and table headers.
ALTER TABLE public.fiscal_configs
    ADD COLUMN IF NOT EXISTS logo        BYTEA,
    ADD COLUMN IF NOT EXISTS logo_mime   VARCHAR(50),
    ADD COLUMN IF NOT EXISTS brand_color VARCHAR(7);
