DROP INDEX IF EXISTS public.idx_fiscal_exits_shipment;
DROP INDEX IF EXISTS public.idx_fiscal_exits_shipment_load;
DROP INDEX IF EXISTS public.idx_fiscal_exits_source_type;

ALTER TABLE public.fiscal_exits
    DROP COLUMN IF EXISTS fiscal_coupon_ecf_serial,
    DROP COLUMN IF EXISTS fiscal_coupon_date,
    DROP COLUMN IF EXISTS fiscal_coupon_number,
    DROP COLUMN IF EXISTS shipment_code,
    DROP COLUMN IF EXISTS shipment_load_code,
    DROP COLUMN IF EXISTS source_type;
