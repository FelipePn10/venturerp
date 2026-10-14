ALTER TABLE public.supplier_scorecard_snapshots DROP CONSTRAINT IF EXISTS supplier_scorecard_snapshots_empresa_periodo_key;
ALTER TABLE public.supplier_scorecard_snapshots
    ADD CONSTRAINT supplier_scorecard_snapshots_supplier_code_period_start_per_key UNIQUE (supplier_code, period_start, period_end);
DROP INDEX IF EXISTS public.ix_supplier_homologations_enterprise;
DROP INDEX IF EXISTS public.ix_receiving_divergences_enterprise;
DROP INDEX IF EXISTS public.ix_receiving_inspection_orders_enterprise;
DROP INDEX IF EXISTS public.ix_procurement_records_enterprise;
ALTER TABLE public.supplier_scorecard_snapshots DROP COLUMN IF EXISTS enterprise_code;
ALTER TABLE public.supplier_homologations       DROP COLUMN IF EXISTS enterprise_code;
ALTER TABLE public.receiving_divergences        DROP COLUMN IF EXISTS enterprise_code;
ALTER TABLE public.receiving_inspection_orders  DROP COLUMN IF EXISTS enterprise_code;
ALTER TABLE public.procurement_records          DROP COLUMN IF EXISTS enterprise_code;
