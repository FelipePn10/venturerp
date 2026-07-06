ALTER TABLE public.fiscal_exits
    ADD COLUMN IF NOT EXISTS source_type VARCHAR(20),
    ADD COLUMN IF NOT EXISTS shipment_load_code BIGINT,
    ADD COLUMN IF NOT EXISTS shipment_code BIGINT,
    ADD COLUMN IF NOT EXISTS fiscal_coupon_number VARCHAR(60),
    ADD COLUMN IF NOT EXISTS fiscal_coupon_date DATE,
    ADD COLUMN IF NOT EXISTS fiscal_coupon_ecf_serial VARCHAR(80);

CREATE INDEX IF NOT EXISTS idx_fiscal_exits_source_type ON public.fiscal_exits(source_type);
CREATE INDEX IF NOT EXISTS idx_fiscal_exits_shipment_load ON public.fiscal_exits(shipment_load_code);
CREATE INDEX IF NOT EXISTS idx_fiscal_exits_shipment ON public.fiscal_exits(shipment_code);
