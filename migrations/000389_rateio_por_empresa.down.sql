DROP INDEX IF EXISTS public.ix_overhead_allocations_enterprise;
ALTER TABLE public.overhead_allocations DROP COLUMN IF EXISTS enterprise_id;
